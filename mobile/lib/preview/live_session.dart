import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';

import 'live_provider.dart';

class LiveSessionFailure implements Exception {
  final String message;
  const LiveSessionFailure(this.message);
  @override
  String toString() => message;
}

// This bounded preview budget is not production authority or accounting.
// Keys exist only in RAM. The ledger stores reservation counters, never content.
class LiveSession extends ChangeNotifier {
  static const sessionId = 'prosody-live-20261101-1';
  static const maxAttempts = 4;
  static const reservedDollarsPerAttempt = 0.25;
  static final Map<String, Future<void>> _ledgerWork = {};
  final File _ledger;
  final LiveBriefingProvider _client;
  Future<void>? _initialization;
  String? _key, _error;
  int _attempts = 0;
  int _configurationRevision = 0;
  final _resultRevisions = Expando<int>('preview result revision');
  bool canPublishResult(LiveBriefing result) =>
      isConfigurationCurrent(_resultRevisions[result] ?? -1);
  int get configurationRevision => _configurationRevision;
  bool isConfigurationCurrent(int revision) =>
      configured && revision == _configurationRevision;
  bool _ready = false, _blocked = false, _busy = false, _disposed = false;

  // Keep the public named argument while the ledger remains private.
  LiveSession({required File ledger, LiveBriefingProvider? client})
    // ignore: prefer_initializing_formals
    : _ledger = ledger,
      _client = client ?? LiveProvider();

  bool get configured => _key != null && _ready && !_blocked && !_disposed;
  bool get busy => _busy;
  int get remainingAttempts =>
      _ready && !_blocked ? maxAttempts - _attempts : 0;
  double get remainingBudgetDollars =>
      remainingAttempts * reservedDollarsPerAttempt;
  String? get error => _error;

  void _notify() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  Future<void> initialize() => _initialization ??= _initialize();

  Future<void> _initialize() async {
    try {
      _attempts = await _ledgerOperation(reserve: false);
      _ready = true;
    } catch (_) {
      _blocked = true;
      _error = 'The saved preview budget is unavailable. Live generation is disabled.';
    }
    _notify();
  }

  void configure(String key) {
    if (_disposed) {
      throw const LiveSessionFailure('This live session has ended.');
    }
    if (key.trim().isEmpty || key.contains('\r') || key.contains('\n')) {
      throw const LiveSessionFailure('A valid API key is required.');
    }
    _configurationRevision++;
    _key = key;
    if (!_blocked) {
      _error = null;
    }
    _notify();
  }

  void clearKey() {
    _configurationRevision++;
    _key = null;
    _notify();
  }

  Future<T> _serialized<T>(Future<T> Function() action) {
    final path = _ledger.absolute.path;
    final previous = _ledgerWork[path] ?? Future<void>.value();
    final result = previous.then((_) async => await action());
    final tail = result.then<void>(
      (_) {},
      onError: (Object _, StackTrace __) {},
    );
    _ledgerWork[path] = tail;
    unawaited(
      tail.then((_) {
        if (identical(_ledgerWork[path], tail)) {
          unawaited(_ledgerWork.remove(path));
        }
      }),
    );
    return result;
  }

  Future<int> _ledgerOperation({
    required bool reserve,
  }) => _serialized(() async {
    await _ledger.parent.create(recursive: true);
    final existed = await _ledger.exists();
    if (reserve && !existed) {
      throw const FormatException('Budget ledger disappeared');
    }
    final file = await _ledger.open(mode: FileMode.append);
    try {
      // No lock retry: uncertainty about another writer fails admission closed.
      await file.lock(FileLock.exclusive);
      final length = await file.length();
      if (length > 4096 || (existed && length == 0)) {
        throw const FormatException('Invalid budget ledger');
      }
      var used = 0;
      if (length > 0) {
        await file.setPosition(0);
        final text = utf8.decode(await file.read(length));
        if (!text.endsWith('\n')) {
          throw const FormatException('Incomplete budget ledger');
        }
        final lines = text.split('\n')..removeLast();
        if (lines.isEmpty || lines.length > maxAttempts + 1) {
          throw const FormatException('Invalid reservation count');
        }
        for (var index = 0; index < lines.length; index++) {
          final value = jsonDecode(lines[index]);
          if (value is! Map<String, dynamic> ||
              value.length != 3 ||
              value['version'] is! int ||
              value['version'] != 1 ||
              value['sessionId'] != sessionId ||
              value['attempts'] is! int ||
              value['attempts'] != index) {
            throw const FormatException('Invalid reservation record');
          }
          used = index;
        }
      }
      if (reserve && used >= maxAttempts) {
        _attempts = used;
        throw const LiveSessionFailure(
          'The authorized preview budget has been fully reserved.',
        );
      }
      if (length == 0 || reserve) {
        if (reserve) {
          used++;
        }
        await file.setPosition(length);
        await file.writeString(
          '${jsonEncode({'version': 1, 'sessionId': sessionId, 'attempts': used})}\n',
        );
        await file.flush();
      }
      return used;
    } finally {
      // Closing also releases the lock. Failure prevents provider dispatch.
      await file.close();
    }
  });

  Future<LiveBriefing> generate(String text) async {
    if (_busy || _disposed) {
      throw const LiveSessionFailure(
        'Live generation is unavailable right now.',
      );
    }
    if (text.trim().isEmpty || text.runes.length > 4000) {
      throw const LiveSessionFailure('Enter between 1 and 4,000 characters.');
    }
    final revision = _configurationRevision;
    final key = _key;
    _busy = true;
    _error = null;
    _notify();
    var reservationStarted = false;
    var reservationCommitted = false;
    try {
      await initialize();
      if (key == null || !isConfigurationCurrent(revision)) {
        throw LiveSessionFailure(
          _error ?? 'Live generation is not configured.',
        );
      }
      if (remainingAttempts == 0) {
        throw const LiveSessionFailure(
          'The authorized preview budget has been fully reserved.',
        );
      }
      final result = await _client.generate(
        key: key,
        source: text,
        reserveAttempt: () async {
          if (reservationStarted || !isConfigurationCurrent(revision)) {
            throw const LiveSessionFailure(
              'This generation cannot be dispatched.',
            );
          }
          reservationStarted = true;
          try {
            _attempts = await _ledgerOperation(reserve: true);
            reservationCommitted = true;
          } on LiveSessionFailure {
            rethrow;
          } catch (_) {
            _blocked = true;
            throw const LiveSessionFailure(
              'The preview budget could not be saved. No further dispatch is allowed.',
            );
          }
          _notify();
          if (!isConfigurationCurrent(revision)) {
            throw const LiveSessionFailure('This live session has ended.');
          }
        },
      );
      if (!isConfigurationCurrent(revision)) {
        throw const LiveSessionFailure(
          'This live session ended. Its result was discarded; the attempt remains reserved.',
        );
      }
      _resultRevisions[result] = revision;
      return result;
    } catch (failure) {
      _error = failure is LiveSessionFailure
          ? failure.message
          : _blocked
          ? 'The preview budget could not be confirmed. Live generation is disabled.'
          : reservationCommitted
          ? 'Generation did not finish. This attempt remains reserved; it will not be retried automatically.'
          : 'Generation could not start. No attempt was reserved. Check your text and live preview setup.';
      throw LiveSessionFailure(_error!);
    } finally {
      _busy = false;
      _notify();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _key = null;
    _client.close();
    super.dispose();
  }
}
