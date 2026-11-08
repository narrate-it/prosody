import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;

import 'bounded_body.dart';
import 'session.dart';

/// Explicit candidate configuration; constructing this is not qualification.
class NativeSessionConfiguration {
  final Uri issuer;
  final String audience;
  static const clientId = 'prosody-ios';
  static const redirect = 'run.prosody.prosody:/oauth/callback';

  /// Requires an explicit HTTPS origin and a nonempty audience.
  NativeSessionConfiguration({required this.issuer, required this.audience}) {
    if (issuer.scheme != 'https' ||
        issuer.host.isEmpty ||
        issuer.userInfo.isNotEmpty ||
        issuer.hasQuery ||
        issuer.hasFragment ||
        issuer.path.isNotEmpty ||
        audience.trim().isEmpty) {
      throw ArgumentError(
        'An exact HTTPS issuer origin and audience are required',
      );
    }
  }

  /// Names the Keychain record by the complete configured client binding.
  String get binding =>
      jsonEncode([issuer.toString(), audience, clientId, redirect]);
}

class NativeSessionFailure implements Exception {
  final String reason;

  /// Stores a diagnostic reason without including grant or callback contents.
  const NativeSessionFailure(this.reason);
  @override
  /// Returns the session failure reason.
  String toString() => 'Native session: $reason';
}

/// Opaque, short-lived personal grants only. No refresh or identity fabrication.
class NativeSession implements InteractiveSession {
  final NativeSessionConfiguration configuration;
  final MethodChannel bridge;
  final http.Client client;
  final DateTime Function() now;
  String? _token;
  DateTime? _expires;
  int _generation = 0;
  bool _busy = false;
  ({String token, DateTime expires})? _pendingRevocation;
  Future<void>? _ending;
  Future<void>? _revoking;

  /// Separate bounded Keychain slot; never usable as an active grant.
  String get _revocationBinding =>
      jsonEncode([configuration.binding, 'pending-revocation.v1']);

  /// Creates a candidate client with injectable platform, transport and clock.
  NativeSession(
    this.configuration, {
    MethodChannel? bridge,
    http.Client? client,
    DateTime Function()? now,
  }) : bridge = bridge ?? const MethodChannel('run.prosody/native_session'),
       client = client ?? http.Client(),
       now = now ?? DateTime.now;

  /// Sends a bounded form request without redirects; ignores successful revoke bodies.
  Future<Map<String, dynamic>> _post(
    String path,
    Map<String, String> body,
  ) async {
    final request = http.Request('POST', configuration.issuer.resolve(path))
      ..followRedirects = false
      ..headers['Accept'] = 'application/json'
      ..bodyFields = {
        ...body,
        'client_id': NativeSessionConfiguration.clientId,
      };
    final response = await client
        .send(request)
        .timeout(const Duration(seconds: 15));
    if (path == '/oauth/revoke' &&
        (response.statusCode == 200 || response.statusCode == 204)) {
      _discardBody(response);
      return {};
    }
    if (response.statusCode != 200) {
      _discardBody(response);
      throw const NativeSessionFailure('Issuer request denied');
    }
    final bytes = await collectBoundedBody(
      response.stream,
      maxBytes: 16384,
      deadline: const Duration(seconds: 15),
    );
    final decoded = jsonDecode(utf8.decode(bytes));
    if (decoded is! Map<String, dynamic>) {
      throw const NativeSessionFailure('Invalid issuer response');
    }
    return decoded;
  }

  /// Requests best-effort cleanup without changing a header-known outcome.
  /// A body or its cancellation can fail or never finish; neither is awaited.
  void _discardBody(http.StreamedResponse response) {
    try {
      unawaited(
        response.stream
            .listen(null, onError: (Object _) {})
            .cancel()
            .catchError((Object _) {}),
      );
    } catch (_) {
      // Cleanup failure cannot reinterpret the already-known HTTP status.
    }
  }

  /// Rejects results belonging to a superseded session operation.
  void _checkGeneration(int generation) {
    if (generation != _generation) {
      throw const NativeSessionFailure('Session changed');
    }
  }

  /// Clears memory immediately, then deletes this binding from Keychain.
  Future<void> _clear() async {
    _token = null;
    _expires = null;
    await bridge.invokeMethod<void>('delete', {
      'binding': configuration.binding,
    });
  }

  /// Loads a recovery target without restoring access or contacting the issuer.
  Future<void> _loadRevocation() async {
    if (_pendingRevocation != null) return;
    final saved = await bridge.invokeMapMethod<String, dynamic>('read', {
      'binding': _revocationBinding,
    });
    if (saved == null) return;
    final token = saved['token'];
    final expiry = DateTime.tryParse(
      saved['expires'] is String ? saved['expires'] as String : '',
    );
    if (token is! String ||
        !RegExp(r'^[A-Za-z0-9_-]{1,128}$').hasMatch(token) ||
        expiry == null) {
      throw const NativeSessionFailure('Invalid sign-out recovery record');
    }
    _pendingRevocation = (token: token, expires: expiry);
  }

  /// Persists the sole target before any path deletes active grant storage.
  Future<void> _prepareRevocation(int generation) async {
    final pending = _pendingRevocation;
    if (pending == null) return;
    _checkGeneration(generation);
    await bridge.invokeMethod<void>('write', {
      'binding': _revocationBinding,
      'token': pending.token,
      'expires': pending.expires.toUtc().toIso8601String(),
    });
    _checkGeneration(generation);
    await _clear();
    _checkGeneration(generation);
  }

  /// Coalesces overlapping recovery requests for the sole pending target.
  Future<void> _revoke(String token) {
    final current = _revoking;
    if (current != null) return current;
    late Future<void> request;
    request = _post('/oauth/revoke', {'token': token})
        .then<void>((_) {})
        .whenComplete(() {
          if (identical(_revoking, request)) _revoking = null;
        });
    _revoking = request;
    return request;
  }

  /// Deletes active storage before one explicit revoke; retains pending on failure.
  Future<void> _finishRevocation(int generation) async {
    final pending = _pendingRevocation;
    if (pending == null) return;
    _checkGeneration(generation);
    await _prepareRevocation(generation);
    await _revoke(pending.token);
    _checkGeneration(generation);
    await bridge.invokeMethod<void>('delete', {'binding': _revocationBinding});
    _checkGeneration(generation);
    if (_pendingRevocation == pending) _pendingRevocation = null;
  }

  /// Checks current personal authority and tightens the local expiry.
  Future<bool> _current(String token, int generation) async {
    final value = await _post('/oauth/current', {'token': token});
    _checkGeneration(generation);
    final expiry = DateTime.tryParse(
      value['expires_at'] is String ? value['expires_at'] as String : '',
    );
    if (value['active'] != true ||
        value['version'] != 'identity.issuer.v1' ||
        value['issuer'] != configuration.issuer.toString() ||
        value['audience'] != configuration.audience ||
        value['client_id'] != NativeSessionConfiguration.clientId ||
        expiry == null ||
        !expiry.isAfter(now()) ||
        _expires == null ||
        !_expires!.isAfter(now())) {
      return false;
    }
    if (generation != _generation) {
      throw const NativeSessionFailure('Session changed');
    }
    if (expiry.isBefore(_expires!)) {
      _expires = expiry;
    }
    return true;
  }

  @override
  /// Restores a persisted unexpired grant only after issuer validation.
  Future<bool> restore() async {
    if (_busy || _ending != null) {
      throw const NativeSessionFailure('Session operation in progress');
    }
    _busy = true;
    try {
      final generation = ++_generation;
      await bridge.invokeMethod<void>('cancel');
      _checkGeneration(generation);
      _token = null;
      _expires = null;
      await _loadRevocation();
      _checkGeneration(generation);
      if (_pendingRevocation != null) {
        await _prepareRevocation(generation);
        return false;
      }
      final saved = await bridge.invokeMapMethod<String, dynamic>('read', {
        'binding': configuration.binding,
      });
      if (generation != _generation) return false;
      if (saved == null) return false;
      final token = saved['token'];
      _expires = DateTime.tryParse(
        saved['expires'] is String ? saved['expires'] as String : '',
      );
      if (token is! String ||
          token.isEmpty ||
          _expires == null ||
          !_expires!.isAfter(now())) {
        await _clear();
        return false;
      }
      try {
        if (!await _current(token, generation)) {
          if (generation == _generation) await _clear();
          return false;
        }
        _token = token;
        return true;
      } catch (_) {
        if (generation == _generation) await _clear();
        rethrow;
      }
    } finally {
      _busy = false;
    }
  }

  @override
  /// Runs private browser return with PKCE and persists a validated personal grant.
  Future<void> signIn() async {
    if (_busy || _ending != null) {
      throw const NativeSessionFailure('Sign-in already in progress');
    }
    _busy = true;
    final generation = ++_generation;
    // Failed recovery discovery must not reopen the previous in-memory grant.
    _token = null;
    _expires = null;
    try {
      await _loadRevocation();
      _checkGeneration(generation);
      await _finishRevocation(generation);
      _checkGeneration(generation);
      await _clear();
      _checkGeneration(generation);
      final proof = await bridge.invokeMapMethod<String, dynamic>('proof');
      _checkGeneration(generation);
      if (proof == null) throw const NativeSessionFailure('PKCE unavailable');
      final verifier = proof['verifier'] as String;
      final state = proof['state'] as String;
      final url = configuration.issuer
          .resolve('/oauth/authorize')
          .replace(
            queryParameters: {
              'response_type': 'code',
              'scope': 'personal',
              'client_id': NativeSessionConfiguration.clientId,
              'redirect_uri': NativeSessionConfiguration.redirect,
              'state': state,
              'code_challenge_method': 'S256',
              'code_challenge': proof['challenge'] as String,
            },
          );
      _checkGeneration(generation);
      final callback = await bridge.invokeMethod<String>('authenticate', {
        'url': url.toString(),
      });
      if (generation != _generation) {
        throw const NativeSessionFailure('Session changed');
      }
      final result = Uri.tryParse(callback ?? '');
      final expected = Uri.parse(NativeSessionConfiguration.redirect);
      if (result == null ||
          result.scheme != expected.scheme ||
          result.hasAuthority ||
          result.path != expected.path ||
          result.hasFragment ||
          result.queryParametersAll['state']?.length != 1 ||
          result.queryParameters['state'] != state ||
          result.queryParametersAll['code']?.length != 1 ||
          result.queryParameters.containsKey('error') ||
          (result.queryParameters['code'] ?? '').isEmpty) {
        throw const NativeSessionFailure('Invalid browser return');
      }
      final started = now();
      final grant = await _post('/oauth/token', {
        'grant_type': 'authorization_code',
        'code': result.queryParameters['code']!,
        'code_verifier': verifier,
        'redirect_uri': NativeSessionConfiguration.redirect,
      });
      final seconds = grant['expires_in'];
      final token = grant['access_token'];
      if (grant['token_type'] != 'Bearer' ||
          grant['scope'] != 'personal' ||
          seconds is! int ||
          seconds < 1 ||
          seconds > 900 ||
          token is! String ||
          !RegExp(r'^[A-Za-z0-9_-]{43}$').hasMatch(token)) {
        throw const NativeSessionFailure('Invalid grant');
      }
      if (generation != _generation) {
        throw const NativeSessionFailure('Session changed');
      }
      _expires = started.add(Duration(seconds: seconds));
      if (!await _current(token, generation)) {
        throw const NativeSessionFailure('Grant is not current');
      }
      await bridge.invokeMethod<void>('write', {
        'binding': configuration.binding,
        'token': token,
        'expires': _expires!.toUtc().toIso8601String(),
      });
      if (generation != _generation) {
        await _clear();
        throw const NativeSessionFailure('Session changed');
      }
      _token = token;
    } finally {
      _busy = false;
    }
  }

  @override
  /// Returns a currently validated grant; rejects pending initialization unchanged.
  Future<String> accessToken() async {
    final token = _token;
    final generation = _generation;
    if (_busy ||
        _ending != null ||
        _pendingRevocation != null ||
        token == null) {
      throw const NativeSessionFailure('Sign-in required');
    }
    if (_expires == null || !_expires!.isAfter(now())) {
      _checkGeneration(generation);
      if (_token == token) {
        await _clear();
      }
      throw const NativeSessionFailure('Sign-in required');
    }
    if (!await _current(token, generation)) {
      _checkGeneration(generation);
      if (_token == token) {
        await _clear();
      }
      throw const NativeSessionFailure('Sign-in required');
    }
    if (generation != _generation ||
        _token != token ||
        !_expires!.isAfter(now())) {
      throw const NativeSessionFailure('Session changed');
    }
    return token;
  }

  @override
  /// Ends local access immediately; explicit retries preserve the original target.
  Future<void> signOut() {
    final current = _ending;
    if (current != null) return current;
    late Future<void> ending;
    ending = _endSession().whenComplete(() {
      if (identical(_ending, ending)) _ending = null;
    });
    _ending = ending;
    return ending;
  }

  /// Persists recovery before active deletion; storage failures remain failures.
  Future<void> _endSession() async {
    final generation = ++_generation;
    final token = _token;
    final expiry = _expires;
    _token = null;
    _expires = null;
    if (token != null && expiry != null) {
      _pendingRevocation ??= (token: token, expires: expiry);
    }
    await _loadRevocation();
    _checkGeneration(generation);
    final pending = _pendingRevocation;
    if (pending != null) {
      await _prepareRevocation(generation);
    }
    await bridge.invokeMethod<void>('cancel');
    _checkGeneration(generation);
    if (pending == null) {
      await _clear();
    } else {
      await _finishRevocation(generation);
    }
  }
}
