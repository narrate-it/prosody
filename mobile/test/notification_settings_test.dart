import 'package:prosody/data/notification_preference.dart';

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/preview/preview_api.dart';
import 'package:prosody/ui/settings.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async =>
      throw StateError('No network in widget tests');
  @override
  Future<void> signOut() async {}
}

FeedSettings _settings(bool enabled) => FeedSettings(
  revision: 7,
  textModel: 'text',
  speechModel: 'speech',
  voice: 'voice',
  hasKey: true,
  notifications: enabled,
);

class _Api extends ProsodyApi {
  _Api()
    : super(
        Uri.parse('https://prosody.example'),
        _Session(),
        client: MockClient(
          (_) => throw StateError('Unexpected network request'),
        ),
      );
  final preferences = <String, bool>{};
  final reads = <String>[];
  final writes = <(String, bool)>[];
  final catalogs = <String>[];
  bool catalogFailure = false;
  Future<List<ProviderModel>> Function(String)? catalog;
  Future<FeedSettings> Function(String)? read;
  Future<void> Function(String, bool)? write;

  @override
  Future<FeedSettings> settings(String feed) async {
    reads.add(feed);
    return read == null
        ? _settings(preferences[feed] ?? false)
        : await read!(feed);
  }

  @override
  Future<void> notifications(String feed, bool enabled) async {
    writes.add((feed, enabled));
    if (write != null) {
      await write!(feed, enabled);
    } else {
      preferences[feed] = enabled;
    }
  }

  @override
  Future<NotificationPreferenceResult> changeNotifications(
    String feed,
    bool enabled, {
    bool Function()? isCurrent,
  }) => confirmNotificationPreference(
    write: () => notifications(feed, enabled),
    read: () => settings(feed),
    // This adapter has no authenticated transport or session token.
    checkSession: () async {},
    isCurrent: isCurrent,
  );

  @override
  Future<List<ProviderModel>> models(String kind) async {
    catalogs.add(kind);
    if (catalog != null) return catalog!(kind);
    if (catalogFailure) throw StateError('private catalog diagnostic');
    return [
      ProviderModel(kind, kind, kind == 'speech' ? ['voice'] : []),
    ];
  }
}

const _personal = Feed(
  id: 'personal',
  name: 'Inbox',
  kind: 'personal',
  canManage: false,
);
const _team = Feed(id: 'team', name: 'Team', kind: 'team', canManage: false);
const _admin = Feed(
  id: 'personal',
  name: 'Inbox',
  kind: 'personal',
  canManage: true,
);

Future<void> _open(
  WidgetTester tester,
  ProsodyApi api, {
  Feed feed = _team,
  bool settle = true,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      home: SettingsScreen(api: api, feed: feed),
    ),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
  }
}

Finder get _settingsScrollable => find
    .descendant(of: find.byType(ListView), matching: find.byType(Scrollable))
    .first;

SwitchListTile _toggle(WidgetTester tester) =>
    tester.widget<SwitchListTile>(find.byType(SwitchListTile));
Future<void> _tap(WidgetTester tester) async {
  await tester.scrollUntilVisible(
    find.text('Arrival notifications'),
    -150,
    scrollable: _settingsScrollable,
  );
  await tester.tap(find.text('Arrival notifications'));
  await tester.pump();
}

void main() {
  for (final status in [401, 403]) {
    for (final transient in [500, 0]) {
      for (final denialFirst in [false, true]) {
        testWidgets(
          'mixed catalog failures clear current draft status=$status transient=$transient denialFirst=$denialFirst',
          (tester) async {
            final text = Completer<List<ProviderModel>>();
            final speech = Completer<List<ProviderModel>>();
            final api = _Api()
              ..catalog = (kind) =>
                  kind == 'text' ? text.future : speech.future;
            addTearDown(api.close);
            await _open(tester, api, feed: _admin);
            await tester.scrollUntilVisible(
              find.byType(TextField),
              150,
              scrollable: _settingsScrollable,
            );
            await tester.enterText(find.byType(TextField), 'synthetic-draft');
            final controller = tester
                .widget<TextField>(find.byType(TextField))
                .controller!;
            final denial = ApiFailure(status, 'private catalog denial');
            final other = transient == 0
                ? StateError('private transport detail')
                : ApiFailure(transient, 'private catalog failure');
            text.completeError(denialFirst ? denial : other);
            await tester.pump();
            expect(controller.text, 'synthetic-draft');
            speech.completeError(denialFirst ? other : denial);
            await tester.pumpAndSettle();
            expect(controller.text, isEmpty);
            expect(api.catalogs, ['text', 'speech']);
            expect(api.writes, isEmpty);
            expect(find.textContaining('private catalog'), findsNothing);
            expect(tester.takeException(), isNull);
          },
        );
      }
    }
  }

  for (final status in [401, 403, 500, 0]) {
    testWidgets('catalog result handles current key draft status=$status', (
      tester,
    ) async {
      final pending = Completer<List<ProviderModel>>();
      final api = _Api()..catalog = (_) => pending.future;
      addTearDown(api.close);
      await _open(tester, api, feed: _admin);
      expect(api.catalogs.length, 2);
      await tester.scrollUntilVisible(
        find.byType(TextField),
        150,
        scrollable: _settingsScrollable,
      );
      await tester.enterText(find.byType(TextField), 'synthetic-draft');
      final controller = tester
          .widget<TextField>(find.byType(TextField))
          .controller!;
      expect(controller.text, 'synthetic-draft');
      pending.completeError(
        status == 0
            ? StateError('private catalog detail')
            : ApiFailure(status, 'private catalog detail'),
      );
      await tester.pumpAndSettle();
      expect(
        controller.text,
        status == 401 || status == 403 ? isEmpty : 'synthetic-draft',
      );
      await tester.scrollUntilVisible(
        find.text('Model choices could not be loaded.'),
        -150,
        scrollable: _settingsScrollable,
      );
      expect(find.text('Model choices could not be loaded.'), findsOneWidget);
      expect(find.textContaining('private catalog detail'), findsNothing);
      await tester.scrollUntilVisible(
        find.text('Arrival notifications'),
        -150,
        scrollable: _settingsScrollable,
      );
      expect(_toggle(tester).onChanged, isNotNull);
      expect(api.reads, ['personal']);
      expect(api.writes, isEmpty);
    });
  }

  for (final status in [401, 403]) {
    testWidgets(
      'stale catalog denial preserves new feed draft status=$status',
      (tester) async {
        final pending = Completer<List<ProviderModel>>();
        var catalogs = 0;
        final api = _Api()
          ..catalog = (kind) {
            if (++catalogs <= 2) return pending.future;
            return Future.value([
              ProviderModel(kind, kind, kind == 'speech' ? ['voice'] : []),
            ]);
          };
        addTearDown(api.close);
        await _open(tester, api, feed: _admin);
        await tester.scrollUntilVisible(
          find.byType(TextField),
          150,
          scrollable: _settingsScrollable,
        );
        await tester.enterText(find.byType(TextField), 'old-synthetic-draft');
        const next = Feed(
          id: 'studio',
          name: 'Studio',
          kind: 'team',
          canManage: true,
        );
        await _open(tester, api, feed: next);
        await tester.scrollUntilVisible(
          find.byType(TextField),
          150,
          scrollable: _settingsScrollable,
        );
        await tester.enterText(find.byType(TextField), 'new-synthetic-draft');
        pending.completeError(ApiFailure(status, 'private old catalog detail'));
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          'new-synthetic-draft',
        );
        expect(find.text('Model choices could not be loaded.'), findsNothing);
        expect(api.reads, ['personal', 'studio']);
        expect(api.catalogs.length, 4);
        expect(tester.takeException(), isNull);
      },
    );
    testWidgets(
      'unmounted catalog denial does not touch disposed key status=$status',
      (tester) async {
        final pending = Completer<List<ProviderModel>>();
        final api = _Api()..catalog = (_) => pending.future;
        addTearDown(api.close);
        await _open(tester, api, feed: _admin);
        await tester.scrollUntilVisible(
          find.byType(TextField),
          150,
          scrollable: _settingsScrollable,
        );
        await tester.enterText(find.byType(TextField), 'synthetic-draft');
        await tester.pumpWidget(const MaterialApp(home: SizedBox()));
        pending.completeError(ApiFailure(status, 'private old catalog detail'));
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        expect(api.writes, isEmpty);
      },
    );
  }

  testWidgets(
    'member preference is off by default and independent of provider catalog',
    (tester) async {
      final api = _Api()..catalogFailure = true;
      addTearDown(api.close);
      await _open(tester, api);
      expect(_toggle(tester).value, isFalse);
      expect(_toggle(tester).onChanged, isNotNull);
      expect(find.text('Save choices'), findsNothing);
      expect(find.text('Model choices could not be loaded.'), findsOneWidget);
      await _tap(tester);
      await tester.pumpAndSettle();
      expect(api.writes, [('team', true)]);
      expect(api.reads, ['team', 'team']);
      expect(_toggle(tester).value, isTrue);
      expect(find.text('Saved for you in this feed.'), findsOneWidget);
      expect(
        find.textContaining('notification permission on your device'),
        findsOneWidget,
      );
      expect(find.textContaining('private catalog'), findsNothing);
    },
  );

  testWidgets(
    'pending write keeps confirmed value and prevents duplicate taps',
    (tester) async {
      final api = _Api();
      final response = Completer<void>();
      api.write = (_, _) => response.future;
      addTearDown(api.close);
      await _open(tester, api);
      final staleCallback = _toggle(tester).onChanged!;
      await _tap(tester);
      expect(_toggle(tester).value, isFalse);
      expect(_toggle(tester).onChanged, isNull);
      staleCallback(true);
      expect(api.writes, [('team', true)]);
      api.preferences['team'] = true;
      response.complete();
      await tester.pumpAndSettle();
      expect(_toggle(tester).value, isTrue);
      expect(api.reads, ['team', 'team']);
    },
  );

  for (final committed in [false, true]) {
    testWidgets(
      'uncertain write reads once without retry committed=$committed',
      (tester) async {
        final api = _Api();
        api.write = (feed, _) async {
          api.preferences[feed] = committed;
          throw StateError('private transport failure');
        };
        addTearDown(api.close);
        await _open(tester, api);
        await _tap(tester);
        await tester.pumpAndSettle();
        expect(api.writes, [('team', true)]);
        expect(api.reads, ['team', 'team']);
        expect(_toggle(tester).value, committed);
        expect(_toggle(tester).onChanged, isNotNull);
        expect(find.text('Current preference confirmed.'), findsOneWidget);
        expect(find.textContaining('private transport'), findsNothing);
      },
    );
  }

  testWidgets(
    'failed readback stays unknown until explicit refresh without new write',
    (tester) async {
      final api = _Api();
      addTearDown(api.close);
      await _open(tester, api);
      api.read = (_) async => throw StateError('private readback failure');
      await _tap(tester);
      await tester.pumpAndSettle();
      expect(_toggle(tester).onChanged, isNull);
      expect(
        find.text(
          'Could not confirm your preference. Refresh to check before changing it.',
        ),
        findsOneWidget,
      );
      expect(find.textContaining('private readback'), findsNothing);
      api.read = null;
      await tester.tap(find.text('Refresh preference'));
      await tester.pumpAndSettle();
      expect(_toggle(tester).value, isTrue);
      expect(_toggle(tester).onChanged, isNotNull);
      expect(api.writes.length, 1);
      expect(api.reads.length, 3);
    },
  );

  for (final status in [401, 403]) {
    testWidgets(
      'authority denial clears preference and stops readback status=$status',
      (tester) async {
        final api = _Api();
        addTearDown(api.close);
        api.write = (_, _) async =>
            throw ApiFailure(status, 'private authority diagnostic');
        await _open(tester, api, feed: _admin);
        await tester.scrollUntilVisible(
          find.byType(TextField),
          150,
          scrollable: _settingsScrollable,
        );
        await tester.enterText(
          find.byType(TextField),
          'synthetic-editor-value',
        );
        await _tap(tester);
        await tester.pumpAndSettle();
        expect(_toggle(tester).onChanged, isNull);
        expect(api.reads, ['personal']);
        expect(find.textContaining('private authority'), findsNothing);
        expect(find.text('Save choices'), findsNothing);
        api.write = null;
        await tester.tap(find.text('Refresh preference'));
        await tester.pumpAndSettle();
        await tester.scrollUntilVisible(
          find.byType(TextField),
          150,
          scrollable: _settingsScrollable,
        );
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          isEmpty,
        );
      },
    );
  }

  testWidgets(
    'feed change invalidates pending write without old-feed readback',
    (tester) async {
      final api = _Api();
      final response = Completer<void>();
      api.write = (_, _) => response.future;
      addTearDown(api.close);
      await _open(tester, api, feed: _personal);
      await _tap(tester);
      await _open(tester, api, feed: _team);
      response.complete();
      await tester.pumpAndSettle();
      expect(api.writes, [('personal', true)]);
      expect(api.reads, ['personal', 'team']);
      expect(_toggle(tester).value, isFalse);
      expect(find.text('Saved for you in this feed.'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('an old rendered callback cannot change a newly selected feed', (
    tester,
  ) async {
    final api = _Api();
    addTearDown(api.close);
    await _open(tester, api, feed: _personal);
    final previous = _toggle(tester).onChanged!;
    await _open(tester, api, feed: _team);
    previous(true);
    await tester.pumpAndSettle();
    expect(api.writes, isEmpty);
    expect(_toggle(tester).value, isFalse);
  });

  testWidgets(
    'API replacement invalidates prior person write even for same feed',
    (tester) async {
      final old = _Api(), next = _Api();
      final response = Completer<void>();
      old.write = (_, _) => response.future;
      addTearDown(old.close);
      addTearDown(next.close);
      await _open(tester, old);
      await _tap(tester);
      await _open(tester, next);
      response.complete();
      await tester.pumpAndSettle();
      expect(old.reads, ['team']);
      expect(next.reads, ['team']);
      expect(_toggle(tester).value, isFalse);
      expect(next.writes, isEmpty);
      expect(find.text('Saved for you in this feed.'), findsNothing);
    },
  );

  testWidgets(
    'stale initial read cannot populate a new feed or fetch old catalogs',
    (tester) async {
      final api = _Api();
      final old = Completer<FeedSettings>();
      api.read = (feed) =>
          feed == 'personal' ? old.future : Future.value(_settings(true));
      addTearDown(api.close);
      await _open(tester, api, feed: _personal, settle: false);
      expect(_toggle(tester).onChanged, isNull);
      await _open(tester, api, feed: _team);
      old.complete(_settings(false));
      await tester.pumpAndSettle();
      expect(_toggle(tester).value, isTrue);
      expect(api.catalogs.length, 2);
      expect(api.reads, ['personal', 'team']);
    },
  );

  testWidgets('unmount after sign-out ignores pending preference completion', (
    tester,
  ) async {
    final api = _Api();
    final response = Completer<void>();
    api.write = (_, _) => response.future;
    addTearDown(api.close);
    await _open(tester, api);
    await _tap(tester);
    await tester.pumpWidget(const MaterialApp(home: SizedBox()));
    response.complete();
    await tester.pumpAndSettle();
    expect(api.reads, ['team']);
    expect(api.writes.length, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('preview preference is explicitly local with no delivery claim', (
    tester,
  ) async {
    final api = PreviewApi(
      durationsMs: {'welcome': 1000, 'design': 1000, 'next': 1000},
    );
    addTearDown(api.close);
    await _open(tester, api, feed: _personal);
    expect(find.textContaining('Local preview only.'), findsOneWidget);
    await _tap(tester);
    await tester.pumpAndSettle();
    expect(_toggle(tester).value, isTrue);
    expect((await api.settings('studio')).notifications, isFalse);
  });
}
