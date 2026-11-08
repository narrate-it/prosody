import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/native_session.dart';

/// Synthetic Keychain map and issuer; not physical-device durability evidence.
class Harness {
  final channel = const MethodChannel('revocation-recovery-test');
  final config = NativeSessionConfiguration(
    issuer: Uri.parse('https://issuer.example.invalid'),
    audience: 'https://api.example.invalid/',
  );
  DateTime clock = DateTime.utc(2026, 11, 8);
  final records = <String, Map<String, dynamic>>{};
  final revoked = <String>[];
  final platformCalls = <String>[];
  int status = 503;
  String? failOperation;
  String? failBinding;
  Completer<void>? holdPendingRead;
  Completer<void>? pendingReadStarted;
  Completer<void>? holdRevoke;
  String get pendingBinding =>
      jsonEncode([config.binding, 'pending-revocation.v1']);

  void install() {
    records[config.binding] = {
      'token': 'a' * 43,
      'expires': clock.add(const Duration(minutes: 10)).toIso8601String(),
    };
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          platformCalls.add(call.method);
          if (call.method == 'cancel') {
            return null;
          }
          if (call.method == 'proof') {
            return {
              'verifier': 'v' * 43,
              'state': 's' * 43,
              'challenge': 'c' * 43,
            };
          }
          if (call.method == 'authenticate') {
            return '${NativeSessionConfiguration.redirect}?code=code&state=${'s' * 43}';
          }
          final binding = (call.arguments as Map)['binding'] as String;
          if (failOperation == call.method &&
              (failBinding == null || failBinding == binding)) {
            throw PlatformException(code: 'synthetic_failure');
          }
          switch (call.method) {
            case 'read':
              final hold = holdPendingRead;
              if (binding == pendingBinding && hold != null) {
                final saved = records[binding];
                final snapshot = saved == null
                    ? null
                    : Map<String, dynamic>.from(saved);
                holdPendingRead = null;
                pendingReadStarted!.complete();
                await hold.future;
                return snapshot;
              }
              return records[binding];
            case 'write':
              records[binding] = Map<String, dynamic>.from(
                call.arguments as Map,
              );
              return null;
            case 'delete':
              records.remove(binding);
              return null;
          }
          throw StateError('Unexpected platform operation');
        });
  }

  NativeSession session() => NativeSession(
    config,
    bridge: channel,
    now: () => clock,
    client: MockClient((request) async {
      if (request.url.path == '/oauth/current') {
        return http.Response(
          jsonEncode({
            'active': true,
            'version': 'identity.issuer.v1',
            'issuer': config.issuer.toString(),
            'audience': config.audience,
            'client_id': NativeSessionConfiguration.clientId,
            'expires_at': clock
                .add(const Duration(minutes: 10))
                .toIso8601String(),
          }),
          200,
        );
      }
      if (request.url.path == '/oauth/revoke') {
        revoked.add(request.bodyFields['token']!);
        await holdRevoke?.future;
        return http.Response('', status);
      }
      throw const NativeSessionFailure('Unexpected token exchange');
    }),
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Harness h;
  setUp(() {
    h = Harness()..install();
  });
  tearDown(
    () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(h.channel, null),
  );

  test(
    'failed sign-in recovery read cannot retain old in-memory access',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      h.failOperation = 'read';
      await expectLater(session.signIn(), throwsA(isA<PlatformException>()));
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(h.revoked, isEmpty);
      // No active storage deletion may precede successful pending-state discovery.
      expect(h.records.containsKey(h.config.binding), isTrue);
    },
  );

  test(
    'invalid sign-in recovery record cannot retain old in-memory access',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      h.records[h.pendingBinding] = {
        'token': 'PRIVATE',
        'expires': 'PRIVATE_INVALID',
      };
      await expectLater(session.signIn(), throwsA(isA<NativeSessionFailure>()));
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(h.revoked, isEmpty);
      expect(h.records.containsKey(h.config.binding), isTrue);
    },
  );

  test(
    'sign-out after failed replacement read revokes original stored grant',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      h.failOperation = 'read';
      await expectLater(session.signIn(), throwsA(isA<PlatformException>()));
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      h.failOperation = null;
      h.status = 204;
      await session.signOut();
      expect(h.revoked, ['a' * 43]);
      expect(h.records, isEmpty);
    },
  );

  test(
    'fresh client sign-out recovers stored target without restoring access',
    () async {
      h.status = 204;
      final session = h.session();
      await session.signOut();
      expect(h.revoked, ['a' * 43]);
      expect(h.records, isEmpty);
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
    },
  );

  test(
    'active fallback read failure preserves stored target for retry',
    () async {
      h.failOperation = 'read';
      h.failBinding = h.config.binding;
      final session = h.session();
      await expectLater(session.signOut(), throwsA(isA<PlatformException>()));
      expect(h.records.containsKey(h.config.binding), isTrue);
      expect(h.records.containsKey(h.pendingBinding), isFalse);
      expect(h.revoked, isEmpty);
      h.failOperation = null;
      h.status = 204;
      await session.signOut();
      expect(h.revoked, ['a' * 43]);
      expect(h.records, isEmpty);
    },
  );

  test('late pending read cannot resurrect completed recovery state', () async {
    h.records[h.pendingBinding] = Map.of(h.records[h.config.binding]!);
    h.holdPendingRead = Completer<void>();
    final hold = h.holdPendingRead!;
    h.pendingReadStarted = Completer<void>();
    final session = h.session();
    final restoring = session.restore();
    final refused = expectLater(
      restoring,
      throwsA(isA<NativeSessionFailure>()),
    );
    await h.pendingReadStarted!.future;
    h.status = 204;
    await session.signOut();
    expect(h.revoked, ['a' * 43]);
    hold.complete();
    await refused;
    await session.signOut();
    expect(h.revoked, ['a' * 43]);
    expect(h.records, isEmpty);
  });

  test('failed revoke explicit retry keeps same original target', () async {
    final session = h.session();
    expect(await session.restore(), isTrue);
    await expectLater(session.signOut(), throwsA(isA<NativeSessionFailure>()));
    await expectLater(
      session.accessToken(),
      throwsA(isA<NativeSessionFailure>()),
    );
    h.status = 204;
    await session.signOut();
    expect(h.revoked, ['a' * 43, 'a' * 43]);
    expect(h.records, isEmpty);
  });

  test('same-session successful signout revokes exactly once', () async {
    h.status = 200;
    final session = h.session();
    expect(await session.restore(), isTrue);
    await session.signOut();
    await session.signOut();
    expect(h.revoked, ['a' * 43]);
    expect(h.records, isEmpty);
  });

  test(
    'new client refuses active restore when recovery record exists',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      await expectLater(
        session.signOut(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(h.records.containsKey(h.pendingBinding), isTrue);
      // Simulate interruption after pending write but before active deletion.
      h.records[h.config.binding] = Map.of(h.records[h.pendingBinding]!);
      final resumed = h.session();
      expect(await resumed.restore(), isFalse);
      await expectLater(
        resumed.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(h.records.containsKey(h.config.binding), isFalse);
      expect(h.revoked.length, 1); // restore never retries a remote request
      h.status = 204;
      await resumed.signOut();
      expect(h.revoked, ['a' * 43, 'a' * 43]);
      expect(h.records, isEmpty);
    },
  );

  test(
    'local expiry never discards an unacknowledged recovery target',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      await expectLater(
        session.signOut(),
        throwsA(isA<NativeSessionFailure>()),
      );
      h.clock = h.clock.add(const Duration(minutes: 11));
      await expectLater(
        session.signOut(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(h.revoked, ['a' * 43, 'a' * 43]);
      expect(h.records.containsKey(h.pendingBinding), isTrue);
      h.status = 204;
      await session.signOut();
      expect(h.revoked.length, 3);
      expect(h.records, isEmpty);
    },
  );

  test('clock rollback cannot prevent explicit recovery', () async {
    final session = h.session();
    expect(await session.restore(), isTrue);
    await expectLater(session.signOut(), throwsA(isA<NativeSessionFailure>()));
    h.clock = h.clock.subtract(const Duration(days: 1));
    final resumed = h.session();
    expect(await resumed.restore(), isFalse);
    h.status = 204;
    await resumed.signOut();
    expect(h.records, isEmpty);
  });

  test('failed initial persistence then sign-in recovery remains restart recoverable', () async {
    final session = h.session();
    expect(await session.restore(), isTrue);
    h.failOperation = 'write';
    await expectLater(session.signOut(), throwsA(isA<PlatformException>()));
    expect(h.records.containsKey(h.config.binding), isTrue);
    h.failOperation = null;
    await expectLater(session.signIn(), throwsA(isA<NativeSessionFailure>()));
    expect(h.records.containsKey(h.pendingBinding), isTrue);
    expect(h.records.containsKey(h.config.binding), isFalse);
    final resumed = h.session();
    expect(await resumed.restore(), isFalse);
    h.status = 204;
    await resumed.signOut();
    expect(h.revoked, ['a' * 43, 'a' * 43]);
    expect(h.records, isEmpty);
  });

  test(
    'restore retries failed persistence before deleting active storage',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      h.failOperation = 'write';
      await expectLater(session.signOut(), throwsA(isA<PlatformException>()));
      await expectLater(session.restore(), throwsA(isA<PlatformException>()));
      expect(h.records.containsKey(h.config.binding), isTrue);
      expect(h.records.containsKey(h.pendingBinding), isFalse);
      h.failOperation = null;
      expect(await session.restore(), isFalse);
      expect(h.records.containsKey(h.config.binding), isFalse);
      expect(h.records.containsKey(h.pendingBinding), isTrue);
      expect(h.revoked, isEmpty);
    },
  );

  test('failed recovery blocks new browser login', () async {
    final session = h.session();
    expect(await session.restore(), isTrue);
    await expectLater(session.signOut(), throwsA(isA<NativeSessionFailure>()));
    h.platformCalls.clear();
    await expectLater(session.signIn(), throwsA(isA<NativeSessionFailure>()));
    expect(h.platformCalls, isNot(contains('proof')));
    expect(h.platformCalls, isNot(contains('authenticate')));
    expect(h.revoked, ['a' * 43, 'a' * 43]);
  });

  test(
    'pending write failure retains local denial and explicit retry target',
    () async {
      final session = h.session();
      expect(await session.restore(), isTrue);
      h.failOperation = 'write';
      await expectLater(session.signOut(), throwsA(isA<PlatformException>()));
      expect(h.revoked, isEmpty);
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      h.failOperation = null;
      h.status = 200;
      await session.signOut();
      expect(h.revoked, ['a' * 43]);
      expect(h.records, isEmpty);
    },
  );

  test('overlapping signout shares one remote revoke', () async {
    final session = h.session();
    expect(await session.restore(), isTrue);
    h.status = 200;
    h.holdRevoke = Completer<void>();
    final first = session.signOut();
    final second = session.signOut();
    h.holdRevoke!.complete();
    await Future.wait([first, second]);
    expect(h.revoked, ['a' * 43]);
  });

  test(
    'corrupt pending record never restores active grant or renders values',
    () async {
      h.records[h.pendingBinding] = {
        'token': 'PRIVATE',
        'expires': 'PRIVATE_INVALID',
      };
      final session = h.session();
      await expectLater(
        session.restore(),
        throwsA(
          isA<NativeSessionFailure>().having(
            (e) => e.toString(),
            'safe diagnostic',
            isNot(contains('PRIVATE')),
          ),
        ),
      );
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(h.revoked, isEmpty);
    },
  );
}
