import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/native_session.dart';

/// Defines isolated native-session contract and concurrency regression tests.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const bridge = MethodChannel('native-session-test');
  final now = DateTime.utc(2026, 11, 4);

  /// Supplies an explicitly nonproduction issuer and audience for test fixtures.
  NativeSessionConfiguration config() => NativeSessionConfiguration(
    issuer: Uri.parse('https://identity.example.invalid'),
    audience: 'test-prosody',
  );
  tearDown(
    () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, null),
  );

  test(
    'configuration requires an exact HTTPS origin and explicit audience',
    () {
      for (final origin in [
        'http://identity.example.invalid',
        'https://user@identity.example.invalid',
        'https://identity.example.invalid/path',
        'https://identity.example.invalid?x=y',
      ]) {
        expect(
          () => NativeSessionConfiguration(
            issuer: Uri.parse(origin),
            audience: 'prosody',
          ),
          throwsArgumentError,
        );
      }
      expect(
        () => NativeSessionConfiguration(
          issuer: Uri.parse('https://identity.example.invalid'),
          audience: '',
        ),
        throwsArgumentError,
      );
    },
  );

  test(
    'expired persisted token is deleted without contacting issuer',
    () async {
      var deleted = false;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(bridge, (call) async {
            if (call.method == 'read' &&
                (call.arguments as Map)['binding'] == config().binding) {
              return {'token': 'expired', 'expires': now.toIso8601String()};
            }
            if (call.method == 'delete') deleted = true;
            return null;
          });
      final session = NativeSession(
        config(),
        bridge: bridge,
        now: () => now,
        client: MockClient(
          (_) async => throw StateError('Must not contact issuer'),
        ),
      );
      expect(await session.restore(), isFalse);
      expect(deleted, isTrue);
    },
  );

  test('restore rejects wrong audience and clears Keychain', () async {
    var deleted = false;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, (call) async {
          if (call.method == 'read' &&
              (call.arguments as Map)['binding'] == config().binding) {
            return {
              'token': 'opaque',
              'expires': now.add(const Duration(minutes: 10)).toIso8601String(),
            };
          }
          if (call.method == 'delete') deleted = true;
          return null;
        });
    final session = NativeSession(
      config(),
      bridge: bridge,
      now: () => now,
      client: MockClient((request) async {
        expect(request.url.path, '/oauth/current');
        expect(request.method, 'POST');
        expect(request.followRedirects, isFalse);
        expect(request.bodyFields, {
          'token': 'opaque',
          'client_id': 'prosody-ios',
        });
        return http.Response(
          jsonEncode({
            'active': true,
            'version': 'identity.issuer.v1',
            'issuer': 'https://identity.example.invalid',
            'client_id': 'prosody-ios',
            'audience': 'another-app',
            'expires_at': now
                .add(const Duration(minutes: 10))
                .toIso8601String(),
          }),
          200,
        );
      }),
    );
    expect(await session.restore(), isFalse);
    expect(deleted, isTrue);
  });

  test('state mismatch never exchanges authorization code', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, (call) async {
          if (call.method == 'proof') {
            return {
              'verifier': 'v' * 43,
              'state': 's' * 43,
              'challenge': 'c' * 43,
            };
          }
          if (call.method == 'authenticate') {
            return 'run.prosody.prosody:/oauth/callback?code=secret&state=wrong';
          }
          return null;
        });
    final session = NativeSession(
      config(),
      bridge: bridge,
      client: MockClient(
        (_) async => throw StateError('Must not exchange code'),
      ),
    );
    await expectLater(session.signIn(), throwsA(isA<NativeSessionFailure>()));
  });
  test(
    'sign-out during pending proof never opens browser or exchanges code',
    () async {
      final proof = Completer<Map<String, dynamic>>();
      final proofStarted = Completer<void>();
      var browsers = 0;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(bridge, (call) async {
            if (call.method == 'proof') {
              proofStarted.complete();
              return proof.future;
            }
            if (call.method == 'authenticate') {
              browsers++;
            }
            return null;
          });
      final session = NativeSession(
        config(),
        bridge: bridge,
        client: MockClient(
          (_) async => throw StateError('No issuer call expected'),
        ),
      );
      final signingIn = session.signIn();
      final rejected = expectLater(
        signingIn,
        throwsA(isA<NativeSessionFailure>()),
      );
      await proofStarted.future;
      await session.signOut();
      proof.complete({
        'verifier': 'v' * 43,
        'state': 's' * 43,
        'challenge': 'c' * 43,
      });
      await rejected;
      expect(browsers, 0);
    },
  );

  for (final scenario in ['normal', 'inactive', 'cancel']) {
    final staleCurrent = scenario == 'inactive';
    final pendingCancel = scenario == 'cancel';
    final name = staleCurrent
        ? 'late inactive current cannot erase a new login'
        : pendingCancel
        ? 'sign-out awaiting cancel blocks login until cleanup completes'
        : 'personal login checks current, stores expiry, and revokes on sign-out';
    test(name, () async {
      Map<String, dynamic>? saved;
      final requests = <http.Request>[];
      final oldCurrent = Completer<http.Response>();
      final oldStarted = Completer<void>();
      final cancelStarted = Completer<void>();
      final cancelResult = Completer<void>();
      var deferCancel = false;
      var deferCurrent = false;
      var issued = 0;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(bridge, (call) async {
            if (call.method == 'cancel' && deferCancel) {
              deferCancel = false;
              cancelStarted.complete();
              await cancelResult.future;
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
              final url = Uri.parse((call.arguments as Map)['url'] as String);
              expect(url.origin, 'https://identity.example.invalid');
              expect(url.queryParameters['client_id'], 'prosody-ios');
              expect(url.queryParameters['code_challenge_method'], 'S256');
              expect(url.queryParameters['scope'], 'personal');
              expect(
                url.queryParameters['redirect_uri'],
                NativeSessionConfiguration.redirect,
              );
              return '${NativeSessionConfiguration.redirect}?code=code&state=${'s' * 43}';
            }
            if (call.method == 'write') {
              saved = Map<String, dynamic>.from(call.arguments as Map);
            }
            if (call.method == 'delete') {
              saved = null;
            }
            return null;
          });
      final session = NativeSession(
        config(),
        bridge: bridge,
        now: () => now,
        client: MockClient((request) async {
          requests.add(request);
          expect(request.followRedirects, isFalse);
          expect(request.bodyFields['client_id'], 'prosody-ios');
          switch (request.url.path) {
            case '/oauth/token':
              expect(request.bodyFields['grant_type'], 'authorization_code');
              expect(
                request.bodyFields['redirect_uri'],
                NativeSessionConfiguration.redirect,
              );
              expect(request.bodyFields['code_verifier'], 'v' * 43);
              issued++;
              return http.Response(
                jsonEncode({
                  'access_token': (issued == 1 ? 'a' : 'b') * 43,
                  'token_type': 'Bearer',
                  'scope': 'personal',
                  'expires_in': 600,
                }),
                200,
              );
            case '/oauth/current':
              if (deferCurrent) {
                deferCurrent = false;
                oldStarted.complete();
                return oldCurrent.future;
              }
              return http.Response(
                jsonEncode({
                  'active': true,
                  'version': 'identity.issuer.v1',
                  'issuer': config().issuer.toString(),
                  'audience': config().audience,
                  'client_id': 'prosody-ios',
                  'expires_at': now
                      .add(const Duration(minutes: 9))
                      .toIso8601String(),
                }),
                200,
              );
            case '/oauth/revoke':
              return http.Response('', 204);
            default:
              throw StateError('Unexpected endpoint');
          }
        }),
      );
      await session.signIn();
      expect(saved?['token'], 'a' * 43);
      expect(
        DateTime.parse(saved!['expires'] as String),
        now.add(const Duration(minutes: 9)),
      );
      expect(await session.accessToken(), 'a' * 43);
      if (staleCurrent) {
        deferCurrent = true;
        final oldRequest = session.accessToken();
        final rejection = expectLater(
          oldRequest,
          throwsA(isA<NativeSessionFailure>()),
        );
        await oldStarted.future;
        await session.signOut();
        await session.signIn();
        expect(saved?['token'], 'b' * 43);
        oldCurrent.complete(http.Response('{"active":false}', 200));
        await rejection;
        expect(saved?['token'], 'b' * 43);
        expect(await session.accessToken(), 'b' * 43);
      }
      if (pendingCancel) {
        deferCancel = true;
        final signingOut = session.signOut();
        await cancelStarted.future;
        await expectLater(
          session.signIn(),
          throwsA(isA<NativeSessionFailure>()),
        );
        cancelResult.complete();
        await signingOut;
        await session.signIn();
        expect(saved?['token'], 'b' * 43);
        expect(await session.accessToken(), 'b' * 43);
        final revocations = requests.where(
          (request) => request.url.path == '/oauth/revoke',
        );
        expect(revocations.last.bodyFields['token'], 'a' * 43);
      }
      await session.signOut();
      expect(saved, isNull);
      expect(requests.last.url.path, '/oauth/revoke');
      expect(
        requests.last.bodyFields['token'],
        (staleCurrent || pendingCancel ? 'b' : 'a') * 43,
      );
    });
  }
  for (final phase in [
    'before restore',
    'restore read',
    'restore current',
    'signIn current',
    'signIn write',
  ]) {
    test('accessToken preserves session during $phase', () async {
      final held = Completer<void>();
      final started = Completer<void>();
      var deletes = 0;
      Map<String, dynamic>? saved = {
        'token': 'a' * 43,
        'expires': now.add(const Duration(minutes: 10)).toIso8601String(),
      };
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(bridge, (call) async {
            if ((phase == 'restore read' &&
                    call.method == 'read' &&
                    (call.arguments as Map)['binding'] == config().binding) ||
                (phase == 'signIn write' && call.method == 'write')) {
              started.complete();
              await held.future;
            }
            if (call.method == 'read' &&
                (call.arguments as Map)['binding'] == config().binding) {
              return saved;
            }
            if (call.method == 'delete') {
              deletes++;
              saved = null;
            }
            if (call.method == 'write') {
              saved = Map<String, dynamic>.from(call.arguments as Map);
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
            return null;
          });
      final session = NativeSession(
        config(),
        bridge: bridge,
        now: () => now,
        client: MockClient((request) async {
          if (request.url.path == '/oauth/token') {
            return http.Response(
              jsonEncode({
                'access_token': 'a' * 43,
                'token_type': 'Bearer',
                'scope': 'personal',
                'expires_in': 600,
              }),
              200,
            );
          }
          if (phase.endsWith('current') && !held.isCompleted) {
            started.complete();
            await held.future;
          }
          return http.Response(
            jsonEncode({
              'active': true,
              'version': 'identity.issuer.v1',
              'issuer': config().issuer.toString(),
              'audience': config().audience,
              'client_id': 'prosody-ios',
              'expires_at': now
                  .add(const Duration(minutes: 9))
                  .toIso8601String(),
            }),
            200,
          );
        }),
      );
      Future<void>? signingIn;
      Future<bool>? restoring;
      if (phase != 'before restore') {
        if (phase.startsWith('restore')) {
          restoring = session.restore();
        } else {
          signingIn = session.signIn();
        }
        await started.future;
      }
      final deletesBeforeRequest = deletes;
      final savedBeforeRequest = saved;
      await expectLater(
        session.accessToken(),
        throwsA(isA<NativeSessionFailure>()),
      );
      expect(deletes, deletesBeforeRequest);
      expect(saved, savedBeforeRequest);
      if (phase == 'before restore') {
        expect(await session.restore(), isTrue);
      } else {
        held.complete();
        if (restoring != null) {
          expect(await restoring, isTrue);
        }
        if (signingIn != null) {
          await signingIn;
        }
      }
      expect(saved?['token'], 'a' * 43);
      expect(await session.accessToken(), 'a' * 43);
    });
  }

  for (final response in [
    (200, ''),
    (200, 'not JSON'),
    (204, ''),
    (401, ''),
    (503, 'unavailable'),
  ]) {
    test(
      'revoke handles status ${response.$1} body ${response.$2} after local clear',
      () async {
        var deleted = false;
        var clock = now;
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(bridge, (call) async {
              if (call.method == 'read' &&
                  (call.arguments as Map)['binding'] == config().binding) {
                return {
                  'token': 'a' * 43,
                  'expires': now
                      .add(const Duration(minutes: 10))
                      .toIso8601String(),
                };
              }
              if (call.method == 'delete') {
                deleted = true;
              }
              return null;
            });
        final session = NativeSession(
          config(),
          bridge: bridge,
          now: () => clock,
          client: MockClient((request) async {
            if (request.url.path == '/oauth/revoke') {
              expect(deleted, isTrue);
              return http.Response(response.$2, response.$1);
            }
            return http.Response(
              jsonEncode({
                'active': true,
                'version': 'identity.issuer.v1',
                'issuer': config().issuer.toString(),
                'audience': config().audience,
                'client_id': 'prosody-ios',
                'expires_at': now
                    .add(const Duration(minutes: 9))
                    .toIso8601String(),
              }),
              200,
            );
          }),
        );
        expect(await session.restore(), isTrue);
        if (response.$1 == 200 || response.$1 == 204) {
          await session.signOut();
        } else {
          await expectLater(
            session.signOut(),
            throwsA(isA<NativeSessionFailure>()),
          );
        }
        expect(deleted, isTrue);
        await expectLater(
          session.accessToken(),
          throwsA(isA<NativeSessionFailure>()),
        );
        // Only acknowledged sign-out permits a separately supplied active record.
        // A failed revoke keeps pending state and must refuse restoration.
        deleted = false;
        expect(
          await session.restore(),
          response.$1 == 200 || response.$1 == 204,
        );
        clock = now.add(const Duration(minutes: 10));
        await expectLater(
          session.accessToken(),
          throwsA(isA<NativeSessionFailure>()),
        );
        expect(deleted, isTrue);
      },
    );
  }
}
