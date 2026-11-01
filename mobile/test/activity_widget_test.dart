import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/ui/activity.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-only';
  @override
  Future<void> signOut() async {}
}

Map<String, Object> item(String id, String status) => {
  'id': id,
  'title': 'Briefing $id',
  'acceptedAt': '2026-11-01T00:00:00Z',
  'status': status,
  'source': 'Original text for $id.\nA second paragraph.',
};

http.Response page(List<Map<String, Object>> items, {String? cursor}) =>
    http.Response(jsonEncode({'items': items, 'nextCursor': cursor}), 200);

void main() {
  ProsodyApi api(Future<http.Response> Function(http.Request) handle) {
    final result = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) {
        // This view must never mutate settings, progress or generation state.
        expect(request.method, 'GET');
        expect(request.url.path, '/prosody/v1/feeds/team/activity');
        return handle(request);
      }),
    );
    addTearDown(result.close);
    return result;
  }

  Future<void> open(WidgetTester tester, ProsodyApi service) async {
    await tester.runAsync(() async {
      await tester.pumpWidget(
        MaterialApp(
          home: ActivityScreen(
            api: service,
            feed: const Feed(
              id: 'team',
              name: 'Team space',
              kind: 'team',
              canManage: false,
            ),
          ),
        ),
      );
      await Future<void>.delayed(Duration.zero);
    });
    await tester.pumpAndSettle();
  }

  Future<void> tapAndLoad(WidgetTester tester, Finder target) async {
    await tester.ensureVisible(target);
    await tester.runAsync(() async {
      await tester.tap(target);
      await Future<void>.delayed(Duration.zero);
    });
    await tester.pumpAndSettle();
  }

  testWidgets('processing and failure remain visible with readable original', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(430, 1200));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    var requests = 0;
    final service = api((_) async {
      requests++;
      return page([item('waiting', 'processing'), item('failed', 'failed')]);
    });
    await open(tester, service);
    expect(find.text('Preparing'), findsOneWidget);
    expect(find.text('Couldn’t prepare briefing'), findsOneWidget);
    expect(
      find.textContaining('A detailed reason is not available.'),
      findsOneWidget,
    );
    expect(find.text('No briefings here yet.'), findsNothing);
    final failed = find.byKey(const ValueKey('activity-failed'));
    await tapAndLoad(
      tester,
      find.descendant(of: failed, matching: find.text('Read original')),
    );
    expect(find.byType(SelectableText), findsOneWidget);
    expect(
      find.text('Original text for failed.\nA second paragraph.'),
      findsOneWidget,
    );
    expect(requests, 1);
    expect(find.text('Retry generation'), findsNothing);
  });

  testWidgets('empty response and failed request have distinct states', (
    tester,
  ) async {
    var fail = true;
    final service = api(
      (_) async => fail
          ? http.Response('private upstream provider error', 500)
          : page([]),
    );
    await open(tester, service);
    expect(find.text('No briefings here yet.'), findsNothing);
    expect(
      find.text('Could not complete this request. Please try again.'),
      findsOneWidget,
    );
    expect(find.textContaining('private upstream'), findsNothing);
    fail = false;
    await tapAndLoad(tester, find.byTooltip('Refresh activity'));
    expect(find.text('No briefings here yet.'), findsOneWidget);
    expect(
      find.text('Could not complete this request. Please try again.'),
      findsNothing,
    );
  });

  testWidgets('pagination is explicit and updates duplicate item status', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(430, 1200));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final cursors = <String?>[];
    final service = api((request) async {
      final cursor = request.url.queryParameters['cursor'];
      cursors.add(cursor);
      if (cursor == null) {
        return page([item('one', 'processing')], cursor: 'second');
      }
      return page([item('one', 'failed'), item('two', 'processing')]);
    });
    await open(tester, service);
    expect(cursors, [null]);
    await tapAndLoad(tester, find.text('Load more activity'));
    expect(cursors, [null, 'second']);
    expect(find.byKey(const ValueKey('activity-one')), findsOneWidget);
    expect(find.text('Couldn’t prepare briefing'), findsOneWidget);
    expect(find.text('Preparing'), findsOneWidget);
    expect(find.text('Load more activity'), findsNothing);
  });

  testWidgets(
    'cyclic pagination reports failure and preserves loaded activity',
    (tester) async {
      var calls = 0;
      final service = api((_) async {
        calls++;
        return page([item('one', 'processing')], cursor: 'cycle');
      });
      await open(tester, service);
      await tapAndLoad(tester, find.text('Load more activity'));
      expect(calls, 2);
      expect(
        find.text('More activity could not be loaded. Please try again.'),
        findsOneWidget,
      );
      expect(find.text('Showing the activity already loaded.'), findsOneWidget);
      expect(find.byKey(const ValueKey('activity-one')), findsOneWidget);
      expect(find.text('No briefings here yet.'), findsNothing);
    },
  );

  testWidgets('late response after disposal does not update screen', (
    tester,
  ) async {
    final response = Completer<http.Response>();
    final service = api((_) => response.future);
    await tester.runAsync(() async {
      await tester.pumpWidget(
        MaterialApp(
          home: ActivityScreen(
            api: service,
            feed: const Feed(
              id: 'team',
              name: 'Team',
              kind: 'team',
              canManage: false,
            ),
          ),
        ),
      );
      await Future<void>.delayed(Duration.zero);
    });
    expect(find.byType(LinearProgressIndicator), findsOneWidget);
    expect(find.text('No briefings here yet.'), findsNothing);
    await tester.pumpWidget(const MaterialApp(home: SizedBox()));
    await tester.runAsync(() async {
      response.complete(page([item('late', 'processing')]));
      await Future<void>.delayed(Duration.zero);
    });
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.text('Briefing late'), findsNothing);
  });
}
