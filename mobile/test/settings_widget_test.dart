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

Finder get _settingsScrollable => find
    .descendant(of: find.byType(ListView), matching: find.byType(Scrollable))
    .first;

void main() {
  Future<void> open(
    WidgetTester tester,
    ProsodyApi api, {
    bool canManage = true,
    String feedId = 'feed',
  }) async {
    await tester.runAsync(() async {
      await tester.pumpWidget(
        MaterialApp(
          home: SettingsScreen(
            api: api,
            feed: Feed(
              id: feedId,
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
    await tester.scrollUntilVisible(
      find.text('Save choices'),
      150,
      scrollable: _settingsScrollable,
    );
    await tester.tap(find.text('Save choices'));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text('Choose an available briefing model.'),
      -150,
      scrollable: _settingsScrollable,
    );
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
    await tester.scrollUntilVisible(
      find.byType(TextField),
      150,
      scrollable: _settingsScrollable,
    );
    await tester.enterText(find.byType(TextField), 'test-provider-key');
    await tester.scrollUntilVisible(
      find.text('Save choices'),
      150,
      scrollable: _settingsScrollable,
    );
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
    await tester.scrollUntilVisible(
      find.text('Briefing model'),
      -150,
      scrollable: _settingsScrollable,
    );
    expect(find.byType(DropdownButtonFormField<String>), findsNWidgets(3));
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

  Future<void> roomy(WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(430, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
  }

  Future<void> enterDraft(WidgetTester tester, String draft) async {
    await tester.scrollUntilVisible(
      find.byType(TextField),
      150,
      scrollable: _settingsScrollable,
    );
    await tester.enterText(find.byType(TextField), draft);
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      draft,
    );
  }

  Future<void> sendSave(WidgetTester tester, {bool settle = true}) async {
    await tester.scrollUntilVisible(
      find.text('Save choices'),
      150,
      scrollable: _settingsScrollable,
    );
    await tester.runAsync(() async {
      await tester.tap(find.text('Save choices'));
      await Future<void>.delayed(Duration.zero);
    });
    if (settle) {
      await tester.pumpAndSettle();
    } else {
      await tester.pump();
    }
  }

  for (final status in [401, 403]) {
    testWidgets('denied settings save clears replacement key status=$status', (
      tester,
    ) async {
      await roomy(tester);
      final writes = <http.Request>[];
      final service = api(
        save: (request) async {
          writes.add(request);
          return http.Response('private denial diagnostic', status);
        },
      );
      await open(tester, service);
      await enterDraft(tester, 'synthetic-denied-draft');
      await sendSave(tester);
      expect(writes.length, 1);
      expect(writes.single.url.path, '/prosody/v1/feeds/feed/settings');
      expect(jsonDecode(writes.single.body), {
        'revision': 7,
        'textModel': 'text',
        'speechModel': 'speech',
        'voice': 'voice',
        'openRouterKey': 'synthetic-denied-draft',
      });
      final field = tester.widget<TextField>(find.byType(TextField));
      expect(field.controller!.text, isEmpty);
      expect(field.enabled, isTrue);
      expect(
        find.text(
          status == 401
              ? 'Your session has expired. Sign in again.'
              : 'You no longer have access to this space.',
        ),
        findsOneWidget,
      );
      expect(find.textContaining('private denial'), findsNothing);
      expect(find.text('Saved for future briefings.'), findsNothing);
      expect(tester.takeException(), isNull);
    });

    testWidgets(
      'late old-feed denial cannot clear new-feed replacement key status=$status',
      (tester) async {
        await roomy(tester);
        late Completer<http.Response> denied;
        await tester.runAsync(() async {
          denied = Completer<http.Response>();
        });
        final writes = <http.Request>[];
        final service = api(
          save: (request) {
            writes.add(request);
            return denied.future;
          },
        );
        await open(tester, service);
        await enterDraft(tester, 'synthetic-old-feed-draft');
        await sendSave(tester, settle: false);
        expect(writes.length, 1);
        expect(writes.single.url.path, '/prosody/v1/feeds/feed/settings');
        // The same API and SettingsScreen state move to a different feed. The
        // existing revision guard must reject the late error from the old scope.
        await open(tester, service, feedId: 'new-feed');
        await enterDraft(tester, 'synthetic-new-feed-draft');
        await tester.runAsync(() async {
          denied.complete(http.Response('private old-feed denial', status));
          await Future<void>.delayed(Duration.zero);
        });
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          'synthetic-new-feed-draft',
        );
        expect(writes.length, 1);
        expect(
          find.text('Your session has expired. Sign in again.'),
          findsNothing,
        );
        expect(
          find.text('You no longer have access to this space.'),
          findsNothing,
        );
        expect(find.textContaining('private old-feed'), findsNothing);
        expect(tester.takeException(), isNull);
      },
    );
  }

  for (final failure in ['409', '500', 'network']) {
    testWidgets(
      'other settings failure retains unsaved replacement key failure=$failure',
      (tester) async {
        await roomy(tester);
        final writes = <http.Request>[];
        final service = api(
          save: (request) async {
            writes.add(request);
            if (failure == 'network') {
              throw http.ClientException('private transport diagnostic');
            }
            return http.Response(
              'private upstream diagnostic',
              int.parse(failure),
            );
          },
        );
        await open(tester, service);
        await enterDraft(tester, 'synthetic-retained-draft');
        await sendSave(tester);
        expect(writes.length, 1);
        expect(writes.single.url.path, '/prosody/v1/feeds/feed/settings');
        expect(
          (jsonDecode(writes.single.body)
              as Map<String, dynamic>)['openRouterKey'],
          'synthetic-retained-draft',
        );
        final field = tester.widget<TextField>(find.byType(TextField));
        expect(field.controller!.text, 'synthetic-retained-draft');
        expect(field.enabled, isTrue);
        expect(
          find.text(switch (failure) {
            '409' => 'This changed on another device. Refresh and try again.',
            '500' => 'Could not complete this request. Please try again.',
            _ => 'Could not save settings.',
          }),
          findsOneWidget,
        );
        expect(find.textContaining('private upstream'), findsNothing);
        expect(find.textContaining('private transport'), findsNothing);
        expect(find.text('Saved for future briefings.'), findsNothing);
        expect(tester.takeException(), isNull);
      },
    );
  }
}
