import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/preview/preview_api.dart';

const _durations = {'welcome': 12000, 'design': 15000, 'next': 13000};
Matcher _status(int status) =>
    isA<ApiFailure>().having((error) => error.status, 'status', status);

void main() {
  test(
    'fixture feeds decode with isolated contents and bundled media IDs',
    () async {
      final api = PreviewApi(durationsMs: _durations);
      addTearDown(api.close);
      final feeds = await api.feeds();
      expect(feeds.map((feed) => feed.name), ['Your inbox', 'Studio']);
      expect(feeds.every((feed) => !feed.canManage), isTrue);
      expect(feeds.map((feed) => feed.kind), ['personal', 'team']);
      final personal = await api.items('personal');
      final studio = await api.items('studio');
      expect(personal.items.map((item) => item.id), ['welcome', 'next']);
      expect(studio.items.map((item) => item.id), ['design']);
      expect(personal.nextCursor, isNull);
      for (final item in [...personal.items, ...studio.items]) {
        expect(item.renditionId, '${item.id}-preview-v1');
        expect(item.segments.single.kind, 'narration');
        expect(
          item.segments.single.media.toString(),
          'https://preview.invalid/${item.id}.mp3',
        );
        expect(item.duration.inMilliseconds, _durations[item.id]);
        expect(item.script, previewNarrationScripts[item.id]);
        expect(item.source, isNotEmpty);
      }
      expect((await api.activity('studio')).items.single.id, 'design');
      final settings = await api.settings('studio');
      expect(settings.hasKey, isFalse);
      expect(settings.notifications, isFalse);
      expect((await api.models('text')).single.id, settings.textModel);
      final speech = (await api.models('speech')).single;
      expect(speech.id, settings.speechModel);
      expect(speech.voices, contains(settings.voice));
      await expectLater(api.session.accessToken(), throwsStateError);
      await api.session.signOut();
    },
  );

  test(
    'progress revisions conflict atomically and mark-read preserves position',
    () async {
      final api = PreviewApi(durationsMs: _durations);
      addTearDown(api.close);
      final item = (await api.items('personal')).items.first;
      final initial = await api.progress(item.id, item.renditionId);
      expect(SavedProgress.fromJson(initial, item).revision, 0);
      final update = <String, dynamic>{
        'renditionId': item.renditionId,
        'expectedRevision': 0,
        'segment': 0,
        'positionMs': 750,
        'played': [
          {'startMs': 0, 'endMs': 500},
        ],
        'viewed': false,
      };
      final first = await api.updateProgress(item.id, update);
      expect(
        SavedProgress.fromJson(
          first,
          item,
          expectedPreviousRevision: 0,
        ).positionMs,
        750,
      );
      await expectLater(
        api.updateProgress(item.id, update),
        throwsA(_status(409)),
      );
      await expectLater(
        api.updateProgress(item.id, {
          ...update,
          'expectedRevision': 1,
          'played': [
            {'startMs': 0, 'endMs': 999999},
          ],
        }),
        throwsA(_status(400)),
      );
      expect((await api.progress(item.id, item.renditionId))['revision'], 1);
      // Returned nested maps are copies, not mutable references into storage.
      ((first['segments'] as List).single as Map)['durationMs'] = 1;
      await api.markRead(item);
      final marked = await api.progress(item.id, item.renditionId);
      expect(marked['markedRead'], isTrue);
      expect(SavedProgress.fromJson(marked, item).positionMs, 750);
      expect(marked['revision'], 2);
      expect(((marked['segments'] as List).single as Map)['coverage'], [
        {'startMs': 0, 'endMs': 500},
      ]);
      expect(
        (await api.progress('design', 'design-preview-v1'))['revision'],
        0,
      );
      await expectLater(
        api.progress('welcome', 'design-preview-v1'),
        throwsA(_status(404)),
      );
    },
  );

  test('credential and unsupported writes never enter preview state', () async {
    final api = PreviewApi(durationsMs: _durations);
    addTearDown(api.close);
    await expectLater(
      api.saveSettings('personal', {'openRouterKey': 'synthetic-not-a-key'}),
      throwsA(_status(403)),
    );
    await expectLater(
      api.request(
        'POST',
        '/prosody/v1/messages/submit-text',
        body: {'text': 'sample'},
      ),
      throwsA(_status(404)),
    );
    await expectLater(
      api.updateProgress('welcome', {'openRouterKey': 'synthetic-not-a-key'}),
      throwsA(_status(400)),
    );
    await expectLater(api.items('unknown'), throwsA(_status(404)));
    await expectLater(
      api.items('personal', cursor: 'unknown'),
      throwsA(_status(400)),
    );
    expect((await api.settings('personal')).hasKey, isFalse);
    expect(
      (await api.progress('welcome', 'welcome-preview-v1'))['revision'],
      0,
    );
  });

  test('local changes disappear with the fixture instance', () async {
    final api = PreviewApi(durationsMs: _durations);
    final other = PreviewApi(durationsMs: _durations);
    addTearDown(other.close);
    await api.deleteItem('welcome');
    expect((await api.items('personal')).items.map((item) => item.id), [
      'next',
    ]);
    expect((await api.activity('personal')).items.map((item) => item.id), [
      'next',
    ]);
    expect((await api.items('studio')).items.single.id, 'design');
    await expectLater(
      api.progress('welcome', 'welcome-preview-v1'),
      throwsA(_status(404)),
    );
    expect((await other.items('personal')).items.map((item) => item.id), [
      'welcome',
      'next',
    ]);
    api.close();
    await expectLater(api.feeds(), throwsA(_status(410)));
  });

  test(
    'missing or invalid measured recording durations fail at construction',
    () {
      for (final durations in <Map<String, int>>[
        {},
        {..._durations, 'welcome': 0},
        {..._durations, 'design': 600001},
      ]) {
        expect(() => PreviewApi(durationsMs: durations), throwsArgumentError);
      }
    },
  );
}
