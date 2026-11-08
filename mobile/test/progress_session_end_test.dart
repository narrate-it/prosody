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
import 'package:prosody/ui/theme.dart';

class _Session implements AccessSession {
  String token = 'old-person';
  bool failOnce = false;
  final ended = Completer<void>();
  @override
  Future<String> accessToken() async => token;
  @override
  Future<void> signOut() async {
    if (failOnce) {
      failOnce = false;
      throw StateError('Synthetic sign-out failure');
    }
    token = 'new-person'; // Models an immediate later session on the same API.
    ended.complete();
  }
}

class _Audio extends Fake implements audio.AudioPlayer {}

class _Playback extends ProsodyPlayback {
  _Playback() : super(audioPlayer: _Audio());
  Briefing? _item;
  bool failStopOnce = false;
  final installed = Completer<void>();
  @override
  Briefing? get current => _item;
  @override
  Duration get position => Duration.zero;
  @override
  Duration get duration => _item?.segments.single.duration ?? Duration.zero;
  @override
  Future<void> replaceFeed(
    List<Briefing> items, {
    int initial = 0,
    int initialSegment = 0,
    Duration initialPosition = Duration.zero,
  }) async {
    _item = items.first;
    location.value = PlaybackLocation(0, 0);
    playbackState.add(
      PlaybackState(
        playing: false,
        processingState: AudioProcessingState.ready,
      ),
    );
    if (!installed.isCompleted) installed.complete();
  }

  @override
  Future<void> stop() async {
    if (failStopOnce) {
      failStopOnce = false;
      throw StateError('Synthetic playback-stop failure');
    }
    _item = null;
    location.value = const PlaybackLocation(0, 0);
  }

  @override
  void appendItems(List<Briefing> items) {}
  @override
  Future<void> dispose() async {
    await stop();
  }
}

Map<String, dynamic> _saved(int revision) => {
  'revision': revision,
  'positionSegment': 0,
  'positionMs': 1000,
  'segments': [
    {'durationMs': 1000},
  ],
};
http.Response _json(Object value) => http.Response(jsonEncode(value), 200);

// Completers synchronize transport handoff. An event turn drains the mock HTTP
// stream's microtasks; no wall-clock sleep stands in for a response signal.
Future<void> _network(
  WidgetTester tester,
  Future<void> Function() action,
) async {
  await tester.runAsync(() async {
    await action();
    await Future<void>(() {});
  });
  await tester.pumpAndSettle();
}

void main() {
  for (final scenario in const [
    (pendingWrite: false, signOut: true, failure: null),
    (pendingWrite: true, signOut: true, failure: null),
    (pendingWrite: false, signOut: false, failure: null),
    (pendingWrite: false, signOut: true, failure: 'stop'),
    (pendingWrite: false, signOut: true, failure: 'sign-out'),
  ]) {
    final pendingWrite = scenario.pendingWrite;
    testWidgets(
      scenario.failure != null
          ? 'failed ${scenario.failure} stays terminal until sign-out retry'
          : scenario.signOut
          ? 'sign-out fences late progress ${pendingWrite ? 'write' : 'read'}'
          : 'same-session teardown still flushes captured progress',
      (tester) async {
        tester.view.physicalSize = const Size(430, 932);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        final session = _Session(), playback = _Playback();
        final reached = Completer<void>(),
            response = Completer<http.Response>();
        final exited = Completer<void>();
        final writeActors = <String>[];
        var reads = 0;
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            if (request.url.path == '/prosody/v1/feeds') {
              return _json({
                'feeds': [
                  {
                    'id': 'shared',
                    'name': 'Shared inbox',
                    'kind': 'team',
                    'canManage': false,
                  },
                ],
              });
            }
            if (request.url.path.endsWith('/items')) {
              return _json({
                'items': [
                  {
                    'id': 'item',
                    'renditionId': 'rendition',
                    'title': 'Briefing',
                    'agent': 'Agent',
                    'status': 'ready',
                    'segments': [
                      {
                        'id': 'audio',
                        'kind': 'audio',
                        'durationMs': 1000,
                        'url': 'https://media.example/audio',
                      },
                    ],
                  },
                ],
                'nextCursor': null,
              });
            }
            if (request.url.path.endsWith('/progress')) {
              if (request.method == 'GET') {
                reads++;
                if (!pendingWrite && reads == 2) {
                  reached.complete();
                  return response.future;
                }
                return _json(_saved(0));
              }
              writeActors.add(request.headers['Authorization'] ?? 'missing');
              final body = jsonDecode(request.body) as Map<String, dynamic>;
              if (pendingWrite && writeActors.length == 1) {
                reached.complete();
                return response.future;
              }
              return _json(_saved((body['expectedRevision'] as int) + 1));
            }
            throw StateError('Unexpected mock route');
          }),
        );
        var signedOut = false;
        try {
          await _network(tester, () async {
            await tester.pumpWidget(
              MaterialApp(
                theme: prosodyTheme(Brightness.light),
                home: StatefulBuilder(
                  builder: (context, setState) => signedOut
                      ? const Scaffold(body: Text('Signed out'))
                      : FeedScreen(
                          api: api,
                          playback: playback,
                          onSignOut: () {
                            setState(() => signedOut = true);
                            exited.complete();
                          },
                        ),
                ),
              ),
            );
          });
          expect(playback.installed.isCompleted, isTrue);
          expect(
            reads,
            1,
          ); // Initial optional resume is a separate completed read.
          await _network(tester, () async {
            playback.completed.value++;
          });
          expect(reached.isCompleted, isTrue);
          if (pendingWrite) {
            // Queue a second snapshot while the first write is already handed off.
            playback.location.value = PlaybackLocation(0, 0);
            playback.completed.value++;
          }
          if (scenario.signOut) {
            playback.failStopOnce = scenario.failure == 'stop';
            session.failOnce = scenario.failure == 'sign-out';
            await _network(tester, () async {
              await tester.tap(find.byTooltip('Sign out'));
            });
            if (scenario.failure != null) {
              expect(exited.isCompleted, isFalse);
              expect(
                find.text('Could not finish signing out.'),
                findsOneWidget,
              );
              expect(find.text('Refresh'), findsNothing);
              final spaceButton = tester.widget<TextButton>(
                find.widgetWithText(TextButton, 'Shared inbox'),
              );
              expect(spaceButton.onPressed, isNull);
              await _network(tester, () async {
                response.complete(_json(_saved(0)));
              });
              expect(writeActors, isEmpty);
              await _network(
                tester,
                () => tester.tap(find.text('Retry sign out')),
              );
            }
            expect(exited.isCompleted, isTrue);
            expect(find.text('Signed out'), findsOneWidget);
            expect(session.token, 'new-person');
          } else {
            await _network(
              tester,
              () => tester.pumpWidget(const SizedBox.shrink()),
            );
            expect(session.token, 'old-person');
          }
          await _network(tester, () async {
            if (!response.isCompleted) {
              response.complete(_json(_saved(pendingWrite ? 1 : 0)));
            }
          });
          expect(
            writeActors,
            pendingWrite || !scenario.signOut ? ['Bearer old-person'] : isEmpty,
            reason: 'Old progress must not initiate a write under the later session',
          );
          expect(reads, 2);
          expect(tester.takeException(), isNull);
        } finally {
          if (!response.isCompleted) {
            response.complete(_json(_saved(pendingWrite ? 1 : 0)));
          }
          await _network(
            tester,
            () => tester.pumpWidget(const SizedBox.shrink()),
          );
          api.close();
          await playback.dispose();
        }
      },
      timeout: const Timeout(Duration(seconds: 10)),
    );
  }
}
