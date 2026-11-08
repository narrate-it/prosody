import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/ui/settings.dart';
import 'package:prosody/preview/preview_api.dart';

class _Session implements AccessSession {
  String token = 'original';
  int reads = 0;
  void Function(int)? onRead;
  @override
  Future<String> accessToken() async {
    final result = token;
    onRead?.call(++reads);
    return result;
  }

  @override
  Future<void> signOut() async {}
}

void main() {
  test('offline preview compound preference stays credential free', () async {
    final api = PreviewApi(
      durationsMs: const {'welcome': 12000, 'design': 15000, 'next': 13000},
    );
    final result = await api.changeNotifications('personal', true);
    expect(result.settings.notifications, isTrue);
    expect(result.writeConfirmed, isTrue);
    await expectLater(
      api.changeNotifications('personal', false, isCurrent: () => false),
      throwsA(isA<ApiFailure>()),
    );
    expect((await api.settings('personal')).notifications, isTrue);
    api.close();
  });
  for (final status in [200, 500, 401, 403]) {
    test('preference write $status stays on the original token', () async {
      final session = _Session();
      final requests = <http.Request>[];
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        session,
        client: MockClient((request) async {
          requests.add(request);
          if (request.method == 'PUT') return http.Response('{}', status);
          return http.Response(
            '{"revision":1,"hasKey":false,"notifications":true}',
            200,
          );
        }),
      );
      if (status == 401 || status == 403) {
        await expectLater(
          api.changeNotifications('feed', true),
          throwsA(isA<ApiFailure>()),
        );
        expect(requests.length, 1);
      } else {
        final result = await api.changeNotifications('feed', true);
        expect(result.settings.notifications, isTrue);
        expect(result.writeConfirmed, status == 200);
        expect(requests.length, 2);
      }
      expect(requests.where((r) => r.method == 'PUT').length, 1);
      expect(
        requests.every((r) => r.headers['Authorization'] == 'Bearer original'),
        isTrue,
      );
      api.close();
    });
  }

  test('preference read failure never repeats the write', () async {
    final requests = <http.Request>[];
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        requests.add(request);
        if (request.method == 'PUT') return http.Response('{}', 200);
        throw StateError('synthetic private transport failure');
      }),
    );
    await expectLater(api.changeNotifications('feed', true), throwsStateError);
    expect(requests.where((r) => r.method == 'PUT').length, 1);
    expect(requests.length, 2);
    api.close();
  });

  test('preference rejects a session change during confirmation', () async {
    final session = _Session();
    final requests = <http.Request>[];
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      session,
      client: MockClient((request) async {
        requests.add(request);
        if (request.method == 'PUT') return http.Response('{}', 200);
        session.token = 'later';
        return http.Response(
          '{"revision":1,"hasKey":false,"notifications":true}',
          200,
        );
      }),
    );
    await expectLater(
      api.changeNotifications('feed', true),
      throwsA(isA<ApiFailure>()),
    );
    expect(
      requests.every((r) => r.headers['Authorization'] == 'Bearer original'),
      isTrue,
    );
    expect(requests.length, 2);
    api.close();
  });

  test(
    'confirmation never borrows a token changed after the local check',
    () async {
      final session = _Session();
      session.onRead = (read) {
        if (read == 2) session.token = 'later';
      };
      final requests = <http.Request>[];
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        session,
        client: MockClient((request) async {
          requests.add(request);
          return http.Response(
            request.method == 'PUT'
                ? '{}'
                : '{"revision":1,"hasKey":false,"notifications":true}',
            200,
          );
        }),
      );
      await expectLater(
        api.changeNotifications('feed', true),
        throwsA(isA<ApiFailure>()),
      );
      expect(requests.length, 2);
      expect(
        requests.every((r) => r.headers['Authorization'] == 'Bearer original'),
        isTrue,
      );
      api.close();
    },
  );

  test(
    'discarded local action suppresses confirmation after its write',
    () async {
      var current = true;
      final requests = <http.Request>[];
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        _Session(),
        client: MockClient((request) async {
          requests.add(request);
          current = false;
          return http.Response('{}', 200);
        }),
      );
      await expectLater(
        api.changeNotifications('feed', true, isCurrent: () => current),
        throwsA(isA<ApiFailure>()),
      );
      expect(requests.length, 1);
      expect(requests.single.method, 'PUT');
      api.close();
    },
  );

  testWidgets('preference reconciliation never reads with a later session', (
    tester,
  ) async {
    final session = _Session();
    final write = Completer<http.Response>();
    final requests = <http.Request>[];
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      session,
      client: MockClient((request) async {
        requests.add(request);
        if (request.method == 'PUT') return write.future;
        if (request.url.path.endsWith('/settings')) {
          return http.Response(
            '{"revision":1,"hasKey":false,"notifications":false}',
            200,
          );
        }
        return http.Response('{"models":[]}', 200);
      }),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          api: api,
          feed: const Feed(
            id: 'feed',
            name: 'Inbox',
            kind: 'personal',
            canManage: false,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.byType(SwitchListTile));
    await tester.tap(find.byType(SwitchListTile));
    await tester.pump();
    expect(requests.where((r) => r.method == 'PUT').length, 1);
    session.token = 'later';
    write.complete(http.Response('{}', 200));
    await tester.pumpAndSettle();
    expect(
      requests.where((r) => r.headers['Authorization'] == 'Bearer later'),
      isEmpty,
    );
    expect(find.text('Saved for you in this feed.'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
    api.close();
  });
}
