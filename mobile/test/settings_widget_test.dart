import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/ui/settings.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-only';
  @override
  Future<void> signOut() async {}
}

void main() {
  Future<void> open(
    WidgetTester tester,
    ProsodyApi api, {
    bool canManage = true,
  }) async {
    await tester.runAsync(() async {
      await tester.pumpWidget(
        MaterialApp(
          home: SettingsScreen(
            api: api,
            feed: Feed(
              id: 'feed',
              name: 'Team',
              kind: 'team',
              canManage: canManage,
            ),
          ),
        ),
      );
      await Future<void>.delayed(Duration.zero);
    });
    await tester.pumpAndSettle();
  }

  ProsodyApi api({
    bool retired = false,
    bool hasKey = true,
    required Future<http.Response> Function(http.Request) save,
  }) {
    final client = MockClient((request) async {
      if (request.method == 'PUT') return save(request);
      if (request.url.path.endsWith('/settings')) {
        return http.Response(
          jsonEncode({
            'revision': 7,
            'textModel': retired ? 'retired' : 'text',
            'speechModel': 'speech',
            'voice': 'voice',
            'hasKey': hasKey,
            'notifications': false,
          }),
          200,
        );
      }
      final speech = request.url.queryParameters['kind'] == 'speech';
      return http.Response(
        jsonEncode({
          'models': [
            {
              'id': speech ? 'speech' : 'text',
              'name': speech ? 'Speech' : 'Text',
              'voices': speech ? ['voice'] : [],
            },
          ],
        }),
        200,
      );
    });
    final result = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: client,
    );
    addTearDown(result.close);
    return result;
  }

  testWidgets('team member cannot edit shared provider settings', (
    tester,
  ) async {
    var writes = 0;
    final service = api(
      save: (_) async {
        writes++;
        return http.Response('{}', 200);
      },
    );
    await open(tester, service, canManage: false);
    expect(find.text('Save choices'), findsNothing);
    expect(find.byType(TextField), findsNothing);
    final dropdowns = tester.widgetList<DropdownButtonFormField<String>>(
      find.byType(DropdownButtonFormField<String>),
    );
    expect(dropdowns.length, 3);
    expect(dropdowns.every((field) => field.onChanged == null), isTrue);
    expect(writes, 0);
    expect(tester.takeException(), isNull);
  });

  testWidgets('retired selection cannot send a settings update', (
    tester,
  ) async {
    var writes = 0;
    final service = api(
      retired: true,
      save: (_) async {
        writes++;
        return http.Response('{}', 200);
      },
    );
    await open(tester, service);
    await tester.ensureVisible(find.text('Save choices'));
    await tester.tap(find.text('Save choices'));
    await tester.pumpAndSettle();
    expect(find.text('Choose an available briefing model.'), findsOneWidget);
    expect(writes, 0);
    expect(tester.takeException(), isNull);
  });

  testWidgets('save binds revision and locks edits until response', (
    tester,
  ) async {
    final response = Completer<http.Response>();
    final writes = <Map<String, dynamic>>[];
    final service = api(
      hasKey: false,
      save: (request) {
        writes.add(jsonDecode(request.body) as Map<String, dynamic>);
        return response.future;
      },
    );
    await open(tester, service);
    await tester.enterText(find.byType(TextField), 'test-provider-key');
    await tester.ensureVisible(find.text('Save choices'));
    await tester.tap(find.text('Save choices'));
    await tester.pump();
    expect(writes, [
      {
        'revision': 7,
        'textModel': 'text',
        'speechModel': 'speech',
        'voice': 'voice',
        'openRouterKey': 'test-provider-key',
      },
    ]);
    expect(tester.widget<TextField>(find.byType(TextField)).enabled, isFalse);
    expect(
      tester.widget<FilledButton>(find.byType(FilledButton)).onPressed,
      isNull,
    );
    expect(
      tester
          .widgetList<DropdownButtonFormField<String>>(
            find.byType(DropdownButtonFormField<String>),
          )
          .every((field) => field.onChanged == null),
      isTrue,
    );
    // Leave while the request is pending: response must not touch disposed UI.
    await tester.pumpWidget(const MaterialApp(home: SizedBox()));
    response.complete(http.Response('{}', 200));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(writes.length, 1);
  });
}
