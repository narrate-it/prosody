import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/preview/preview_api.dart';
import 'package:prosody/ui/settings.dart';

class _Session implements AccessSession {
  String? token = 'original';
  @override
  Future<String> accessToken() async {
    if (token == null) throw const ApiFailure(401, 'Session expired.');
    return token!;
  }

  @override
  Future<void> signOut() async {}
}

const _body = '{"revision":1,"hasKey":false,"notifications":true}';

void main() {
  for (final change in ['replacement', 'expiry']) {
    test('settings read refuses $change during response', () async {
      final session = _Session();
      var calls = 0;
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        session,
        client: MockClient((request) async {
          calls++;
          expect(request.headers['Authorization'], 'Bearer original');
          session.token = change == 'expiry' ? null : 'later';
          return http.Response(_body, 200);
        }),
      );
      await expectLater(api.settings('feed'), throwsA(isA<ApiFailure>()));
      expect(calls, 1);
      api.close();
    });
  }

  test('expired initial session sends no settings request', () async {
    final session = _Session()..token = null;
    var calls = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      session,
      client: MockClient((request) async {
        calls++;
        return http.Response(_body, 200);
      }),
    );
    await expectLater(api.settings('feed'), throwsA(isA<ApiFailure>()));
    expect(calls, 0);
    api.close();
  });

  test('settings denial retains its status without retry', () async {
    var calls = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        calls++;
        return http.Response('private diagnostic', 403);
      }),
    );
    await expectLater(
      api.settings('feed'),
      throwsA(
        isA<ApiFailure>()
            .having((e) => e.status, 'status', 403)
            .having(
              (e) => e.message.contains('private diagnostic'),
              'private body',
              false,
            ),
      ),
    );
    expect(calls, 1);
    api.close();
  });

  test('stable settings read retains fields and uses one request', () async {
    var calls = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        calls++;
        expect(request.headers['Authorization'], 'Bearer original');
        return http.Response(_body, 200);
      }),
    );
    final result = await api.settings('feed');
    expect(result.revision, 1);
    expect(result.notifications, isTrue);
    expect(result.hasKey, isFalse);
    expect(calls, 1);
    api.close();
  });

  test('offline preview settings remain credential free', () async {
    final api = PreviewApi(
      durationsMs: const {'welcome': 12000, 'design': 15000, 'next': 13000},
    );
    expect((await api.settings('personal')).revision, 0);
    api.close();
  });

  testWidgets(
    'old-session settings do not start catalogs or enable preferences',
    (tester) async {
      final session = _Session();
      final held = Completer<http.Response>();
      final requests = <http.Request>[];
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        session,
        client: MockClient((request) async {
          requests.add(request);
          if (request.url.path.endsWith('/settings')) return held.future;
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
              canManage: true,
            ),
          ),
        ),
      );
      await tester.pump();
      expect(requests.length, 1);
      session.token = 'later';
      held.complete(http.Response(_body, 200));
      await tester.pumpAndSettle();
      expect(requests.length, 1);
      expect(
        tester.widget<SwitchListTile>(find.byType(SwitchListTile)).onChanged,
        isNull,
      );
      await tester.pumpWidget(const SizedBox.shrink());
      api.close();
    },
  );
}
