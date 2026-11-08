import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/native_session.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const bridge = MethodChannel('native-revoke-body-test');
  final now = DateTime.utc(2026, 11, 8);
  final config = NativeSessionConfiguration(
    issuer: Uri.parse('https://identity.example.invalid'),
    audience: 'test-prosody',
  );
  tearDown(
    () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, null),
  );

  Future<(NativeSession, List<String>, bool Function())> restored(
    http.StreamedResponse response,
  ) async {
    var deleted = false;
    final paths = <String>[];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, (call) async {
          if (call.method == 'read') {
            return {
              'token': 'a' * 43,
              'expires': now.add(const Duration(minutes: 10)).toIso8601String(),
            };
          }
          if (call.method == 'delete') deleted = true;
          return null;
        });
    final session = NativeSession(
      config,
      bridge: bridge,
      now: () => now,
      client: MockClient.streaming((request, body) async {
        paths.add(request.url.path);
        if (request.url.path == '/oauth/revoke') {
          expect(deleted, isTrue);
          return response;
        }
        expect(request.url.path, '/oauth/current');
        return http.StreamedResponse(
          Stream.value(
            utf8.encode(
              jsonEncode({
                'active': true,
                'version': 'identity.issuer.v1',
                'issuer': config.issuer.toString(),
                'audience': config.audience,
                'client_id': 'prosody-ios',
                'expires_at': now
                    .add(const Duration(minutes: 9))
                    .toIso8601String(),
              }),
            ),
          ),
          200,
        );
      }),
    );
    expect(await session.restore(), isTrue);
    return (session, paths, () => deleted);
  }

  for (final status in [200, 204]) {
    for (final kind in ['oversized', 'error', 'empty']) {
      test('acknowledged revoke $status ignores $kind body', () async {
        final stream = switch (kind) {
          'oversized' => Stream<List<int>>.value(List.filled(16385, 1)),
          'error' => Stream<List<int>>.error(
            StateError('private revocation body'),
          ),
          _ => const Stream<List<int>>.empty(),
        };
        final (session, paths, deleted) = await restored(
          http.StreamedResponse(stream, status),
        );
        addTearDown(session.client.close);
        await expectLater(session.signOut(), completes);
        expect(deleted(), isTrue);
        await expectLater(
          session.accessToken(),
          throwsA(isA<NativeSessionFailure>()),
        );
        expect(paths, ['/oauth/current', '/oauth/revoke']);
      });
    }
  }

  for (final status in [401, 503]) {
    test(
      'denied revoke $status preserves safe outcome despite body failure',
      () async {
        final (session, paths, deleted) = await restored(
          http.StreamedResponse(
            Stream<List<int>>.error(StateError('private revocation body')),
            status,
          ),
        );
        addTearDown(session.client.close);
        await expectLater(
          session.signOut(),
          throwsA(
            isA<NativeSessionFailure>().having(
              (e) => e.reason,
              'reason',
              'Issuer request denied',
            ),
          ),
        );
        expect(deleted(), isTrue);
        await expectLater(
          session.accessToken(),
          throwsA(isA<NativeSessionFailure>()),
        );
        expect(paths, ['/oauth/current', '/oauth/revoke']);
      },
    );
  }
  test('acknowledged revoke does not await unfinished body cleanup', () async {
    final cleanup = Completer<void>();
    var cancelled = false;
    final body = StreamController<List<int>>(
      onCancel: () {
        cancelled = true;
        return cleanup.future;
      },
    );
    final (session, paths, deleted) = await restored(
      http.StreamedResponse(body.stream, 200),
    );
    addTearDown(session.client.close);
    try {
      // Bounded harness control, not a real-network latency qualification.
      await session.signOut().timeout(const Duration(seconds: 5));
      expect(cancelled, isTrue);
      expect(cleanup.isCompleted, isFalse);
      expect(deleted(), isTrue);
      expect(paths, ['/oauth/current', '/oauth/revoke']);
    } finally {
      cleanup.complete();
      await body.close();
    }
  });

  for (final status in [401, 503]) {
    test(
      'current denial $status refuses before broken body diagnostics',
      () async {
        var deleted = false;
        var calls = 0;
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(bridge, (call) async {
              if (call.method == 'read') {
                return {
                  'token': 'a' * 43,
                  'expires': now
                      .add(const Duration(minutes: 10))
                      .toIso8601String(),
                };
              }
              if (call.method == 'delete') deleted = true;
              return null;
            });
        final session = NativeSession(
          config,
          bridge: bridge,
          now: () => now,
          client: MockClient.streaming((request, body) async {
            calls++;
            expect(request.url.path, '/oauth/current');
            return http.StreamedResponse(
              Stream<List<int>>.error(StateError('private current body')),
              status,
            );
          }),
        );
        addTearDown(session.client.close);
        await expectLater(
          session.restore(),
          throwsA(
            isA<NativeSessionFailure>().having(
              (e) => e.reason,
              'reason',
              'Issuer request denied',
            ),
          ),
        );
        expect(calls, 1);
        expect(deleted, isTrue);
      },
    );
  }

  for (final kind in ['oversized', 'error']) {
    test(
      'successful current still requires a bounded valid $kind body',
      () async {
        var deleted = false;
        var calls = 0;
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(bridge, (call) async {
              if (call.method == 'read') {
                return {
                  'token': 'a' * 43,
                  'expires': now
                      .add(const Duration(minutes: 10))
                      .toIso8601String(),
                };
              }
              if (call.method == 'delete') deleted = true;
              return null;
            });
        final session = NativeSession(
          config,
          bridge: bridge,
          now: () => now,
          client: MockClient.streaming((request, body) async {
            calls++;
            expect(request.url.path, '/oauth/current');
            return http.StreamedResponse(
              kind == 'oversized'
                  ? Stream<List<int>>.value(List.filled(16385, 1))
                  : Stream<List<int>>.error(StateError('private current body')),
              200,
            );
          }),
        );
        addTearDown(session.client.close);
        await expectLater(session.restore(), throwsA(anything));
        expect(calls, 1);
        expect(deleted, isTrue);
      },
    );
  }
}
