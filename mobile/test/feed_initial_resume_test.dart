import 'dart:async';
import 'dart:convert';
import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/playback/handler.dart';
import 'package:prosody/ui/feed.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-only';
  @override
  Future<void> signOut() async {}
}

class _Audio extends Fake implements audio.AudioPlayer {}

// Records the UI-to-player initialization boundary without native decoders.
class _Playback extends ProsodyPlayback {
  _Playback() : super(audioPlayer: _Audio());
  Briefing? _current;
  List<Briefing> _items = [];
  bool allowManualSelection = false;
  final manualSelections = <int>[];
  Duration _position = Duration.zero;
  final starts = <({String id, int segment, Duration position})>[];
  int appends = 0;
  void playAt(Duration position) {
    _position = position;
    playbackState.add(
      PlaybackState(
        processingState: AudioProcessingState.ready,
        playing: true,
      ),
    );
  }

  @override
  Briefing? get current => _current;
  @override
  Duration get position => _position;
  @override
  Future<void> stop() async {
    _current = null;
    _items = [];
    _position = Duration.zero;
    location.value = const PlaybackLocation(0, 0);
  }

  @override
  Future<void> replaceFeed(
    List<Briefing> items, {
    int initial = 0,
    int initialSegment = 0,
    Duration initialPosition = Duration.zero,
  }) async {
    await stop();
    if (items.isEmpty) return;
    _items = List.of(items);
    _current = items[initial];
    _position = initialPosition;
    starts.add((
      id: _current!.id,
      segment: initialSegment,
      position: initialPosition,
    ));
    location.value = PlaybackLocation(initial, initialSegment);
    playbackState.add(
      PlaybackState(
        processingState: AudioProcessingState.ready,
        playing: false,
      ),
    );
  }

  @override
  void appendItems(List<Briefing> items) {
    _items.addAll(items);
    appends++;
  }

  @override
  Future<void> select(
    int index, {
    int segment = 0,
    bool autoplay = true,
  }) async {
    if (!allowManualSelection) {
      fail('Initial playback must restore before selecting/autoplay');
    }
    manualSelections.add(index);
    _current = _items[index];
    _position = Duration.zero;
    location.value = PlaybackLocation(index, segment);
  }

  @override
  Future<void> dispose() async {
    await stop();
  }
}

Map<String, dynamic> _item(String id) => {
  'id': id,
  'renditionId': '$id-r',
  'title': 'Briefing $id',
  'agent': 'Agent',
  'status': 'ready',
  'segments': [
    {
      'id': 'one',
      'kind': 'audio',
      'durationMs': 1000,
      'url': 'https://media.example/one',
    },
    {
      'id': 'two',
      'kind': 'audio',
      'durationMs': 1000,
      'url': 'https://media.example/two',
    },
  ],
};
http.Response _json(Object value) => http.Response(jsonEncode(value), 200);
http.Response _saved() => _json({
  'revision': 4,
  'positionSegment': 1,
  'positionMs': 400,
  'segments': [
    {'durationMs': 1000},
    {'durationMs': 1000},
  ],
});

http.Response _acknowledgeProgress(http.Request request) {
  final update = jsonDecode(request.body) as Map<String, dynamic>;
  return _json({
    'revision': (update['expectedRevision'] as int) + 1,
    'positionSegment': update['segment'],
    'positionMs': update['positionMs'],
    'segments': [
      {'durationMs': 1000},
      {'durationMs': 1000},
    ],
  });
}

Future<void> _flush(WidgetTester tester, Future<void> Function() action) async {
  await tester.runAsync(() async {
    await action();
    await Future<void>.delayed(const Duration(milliseconds: 20));
  });
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'switching from page three shows restored first item of new space',
    (tester) async {
      final playback = _Playback()..allowManualSelection = true;
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        _Session(),
        client: MockClient((request) async {
          if (request.method == 'PUT') return _acknowledgeProgress(request);
          final path = request.url.path;
          if (path == '/prosody/v1/feeds') {
            return _json({
              'feeds': [
                for (final id in ['a', 'b'])
                  {
                    'id': id,
                    'name': id.toUpperCase(),
                    'kind': 'team',
                    'canManage': false,
                  },
              ],
            });
          }
          if (path.endsWith('/items')) {
            final prefix = path.contains('/a/') ? 'a' : 'b';
            return _json({
              'items': [
                _item('$prefix-item'),
                _item('$prefix-two'),
                _item('$prefix-three'),
              ],
              'nextCursor': null,
            });
          }
          return _saved();
        }),
      );
      try {
        await _flush(
          tester,
          () => tester.pumpWidget(
            MaterialApp(
              home: FeedScreen(api: api, playback: playback, onSignOut: () {}),
            ),
          ),
        );
        tester
            .widget<PageView>(find.byType(PageView))
            .controller!
            .jumpToPage(2);
        await tester.pumpAndSettle();
        expect(playback.current?.id, 'a-three');
        expect(playback.manualSelections, [2]);
        await _flush(tester, () => tester.tap(find.text('A')));
        await _flush(tester, () => tester.tap(find.text('B')));
        expect(
          tester.widget<PageView>(find.byType(PageView)).controller!.page,
          0,
        );
        expect(find.text('Briefing b-item').hitTestable(), findsOneWidget);
        expect(playback.current?.id, 'b-item');
        expect(playback.starts.last.segment, 1);
        expect(
          playback.starts.last.position,
          const Duration(milliseconds: 400),
        );
        expect(
          playback.manualSelections,
          [2],
          reason: 'Feed reset must not select a stale page',
        );
      } finally {
        await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
        api.close();
        await playback.dispose();
      }
    },
  );

  testWidgets(
    'stop callbacks cannot prefetch the old space under the new revision',
    (tester) async {
      final playback = _Playback();
      var oldPages = 0, oldReads = 0, newReads = 0;
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        _Session(),
        client: MockClient((request) async {
          if (request.method == 'PUT') return _acknowledgeProgress(request);
          expect(request.method, 'GET');
          final path = request.url.path;
          if (path == '/prosody/v1/feeds') {
            return _json({
              'feeds': [
                for (final id in ['a', 'b'])
                  {
                    'id': id,
                    'name': id.toUpperCase(),
                    'kind': 'team',
                    'canManage': false,
                  },
              ],
            });
          }
          if (path == '/prosody/v1/feeds/a/items') {
            oldPages++;
            if (!request.url.hasQuery) {
              return _json({
                'items': [_item('a-item'), _item('a-two'), _item('a-three')],
                'nextCursor': 'later',
              });
            }
            expect(
              request.url.queryParameters['cursor'],
              'later',
              reason: 'Stop must not launch an old-space later2 request',
            );
            return _json({'items': [], 'nextCursor': 'later2'});
          }
          if (path == '/prosody/v1/feeds/b/items') {
            return _json({
              'items': [_item('b-item')],
              'nextCursor': null,
            });
          }
          if (path.contains('/a-item/')) {
            oldReads++;
          } else {
            newReads++;
          }
          return _saved();
        }),
      );
      try {
        await _flush(
          tester,
          () => tester.pumpWidget(
            MaterialApp(
              home: FeedScreen(api: api, playback: playback, onSignOut: () {}),
            ),
          ),
        );
        expect(oldPages, 2);
        expect(
          oldReads,
          1,
          reason: 'Appending to an active queue must not restore again',
        );
        expect(playback.appends, 1);
        await _flush(tester, () => tester.tap(find.text('A')));
        await _flush(tester, () => tester.tap(find.text('B')));
        expect(oldPages, 2);
        expect(newReads, 1);
        expect(playback.current?.id, 'b-item');
      } finally {
        await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
        api.close();
        await playback.dispose();
      }
    },
  );

  for (final mode in ['saved', 'unknown', '401', '403']) {
    testWidgets('first playable later page initializes resume: $mode', (
      tester,
    ) async {
      final playback = _Playback();
      var reads = 0, writes = 0;
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        _Session(),
        client: MockClient((request) async {
          if (request.method != 'GET') {
            writes++;
            expect(
              mode,
              'saved',
              reason: 'Unknown/denied resume must not write',
            );
            return _acknowledgeProgress(request);
          }
          if (request.url.path == '/prosody/v1/feeds') {
            return _json({
              'feeds': [
                {'id': 'a', 'name': 'A', 'kind': 'personal', 'canManage': true},
              ],
            });
          }
          if (request.url.path.endsWith('/items')) {
            return _json({
              'items': request.url.hasQuery ? [_item('a-item')] : [],
              'nextCursor': request.url.hasQuery ? null : 'later',
            });
          }
          expect(request.url.path, '/prosody/v1/items/a-item/progress');
          reads++;
          if (mode == 'saved') return _saved();
          return http.Response('private error', int.tryParse(mode) ?? 503);
        }),
      );
      try {
        await _flush(
          tester,
          () => tester.pumpWidget(
            MaterialApp(
              home: FeedScreen(api: api, playback: playback, onSignOut: () {}),
            ),
          ),
        );
        expect(reads, 1);
        if (mode == 'saved') {
          expect(playback.starts.single.segment, 1);
          expect(
            playback.starts.single.position,
            const Duration(milliseconds: 400),
          );
        } else if (mode == 'unknown') {
          expect(playback.starts.single.segment, 0);
        } else {
          expect(playback.starts, isEmpty);
          expect(playback.current, isNull);
        }
        expect(writes, 0);
        if (mode == 'saved' || mode == 'unknown') {
          // Deliver the active position before completion so the known-resume
          // control actually dirties and dispatches a recorder update.
          await tester.runAsync(() async {
            playback.playAt(const Duration(milliseconds: 400));
            await Future<void>.delayed(const Duration(milliseconds: 20));
            playback.completed.value++;
            await Future<void>.delayed(const Duration(milliseconds: 20));
          });
          // The active waveform deliberately never settles.
          await tester.pump();
          expect(reads, mode == 'saved' ? 2 : 1);
          expect(writes, mode == 'saved' ? 1 : 0);
          if (mode == 'unknown') {
            expect(
              find.textContaining('saved place could not be loaded'),
              findsOneWidget,
            );
          }
        }
      } finally {
        await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
        api.close();
        await playback.dispose();
      }
    });
  }

  testWidgets('stale later-page resume cannot alter the newly selected space', (
    tester,
  ) async {
    final pending = Completer<http.Response>();
    final playback = _Playback();
    var oldReads = 0, newReads = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        if (request.method == 'PUT') return _acknowledgeProgress(request);
        expect(request.method, 'GET');
        final path = request.url.path;
        if (path == '/prosody/v1/feeds') {
          return _json({
            'feeds': [
              for (final id in ['a', 'b'])
                {
                  'id': id,
                  'name': id.toUpperCase(),
                  'kind': 'team',
                  'canManage': false,
                },
            ],
          });
        }
        if (path.endsWith('/items')) {
          final isNew = path.contains('/b/');
          return _json({
            'items': isNew || request.url.hasQuery
                ? [_item(isNew ? 'b-item' : 'a-item')]
                : [],
            'nextCursor': isNew || request.url.hasQuery ? null : 'later',
          });
        }
        if (path.contains('/a-item/')) {
          oldReads++;
          return pending.future;
        }
        newReads++;
        return _saved();
      }),
    );
    try {
      await _flush(
        tester,
        () => tester.pumpWidget(
          MaterialApp(
            home: FeedScreen(api: api, playback: playback, onSignOut: () {}),
          ),
        ),
      );
      expect(oldReads, 1);
      await _flush(tester, () => tester.tap(find.text('A')));
      await _flush(tester, () => tester.tap(find.text('B')));
      expect(playback.starts.single.id, 'b-item');
      await _flush(tester, () async {
        pending.complete(http.Response('unavailable', 503));
      });
      expect(playback.starts.single.id, 'b-item');
      expect(newReads, 1);
      expect(
        find.textContaining('saved place could not be loaded'),
        findsNothing,
      );
    } finally {
      if (!pending.isCompleted) {
        pending.complete(http.Response('unavailable', 503));
      }
      await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
      api.close();
      await playback.dispose();
    }
  });
}
