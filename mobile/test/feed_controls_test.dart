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
import 'package:prosody/ui/activity.dart';
import 'package:prosody/ui/feed.dart';
import 'package:prosody/ui/settings.dart';
import 'package:prosody/ui/theme.dart';

class _Session implements AccessSession {
  int signOuts = 0;
  Completer<void>? signOutGate;
  @override
  Future<String> accessToken() async => 'test-only';
  @override
  Future<void> signOut() async {
    signOuts++;
    await signOutGate?.future;
  }
}

class _Audio extends Fake implements audio.AudioPlayer {}

class _Playback extends ProsodyPlayback {
  _Playback() : super(audioPlayer: _Audio());
  Briefing? _current;
  Duration _position = Duration.zero;
  final seeks = <Duration>[];
  int stops = 0, starts = 0;
  @override
  Briefing? get current => _current;
  @override
  Duration get position => _position;
  @override
  Duration get duration =>
      _current?.segments[location.value.segment].duration ?? Duration.zero;
  void transitionToSegment(int index, int seconds) {
    _position = Duration(seconds: seconds);
    location.value = PlaybackLocation(0, index);
  }

  void setPosition(int seconds) {
    _position = Duration(seconds: seconds);
    playbackState.add(
      PlaybackState(
        processingState: AudioProcessingState.ready,
        playing: false,
        updatePosition: _position,
      ),
    );
  }

  @override
  Future<void> seekSegment(Duration position) async {
    seeks.add(position);
    discontinuity.value++;
    setPosition(position.inSeconds);
  }

  @override
  Future<void> stop() async {
    stops++;
    _current = null;
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
    if (items.isEmpty) return;
    starts++;
    _current = items[initial];
    location.value = PlaybackLocation(initial, initialSegment);
    setPosition(initialPosition.inSeconds);
  }

  @override
  void appendItems(List<Briefing> items) {}
  @override
  Future<void> dispose() async {
    await stop();
  }
}

http.Response _json(Object body) => http.Response(jsonEncode(body), 200);

ProsodyApi _api(
  _Session session, {
  bool shortThenLong = false,
  Future<http.Response>? directory,
}) => ProsodyApi(
  Uri.parse('https://prosody.example'),
  session,
  client: MockClient((request) async {
    final path = request.url.path;
    if (path == '/prosody/v1/feeds') {
      if (directory != null) return directory;
      return _json({
        'feeds': [
          {
            'id': 'personal',
            'name': 'Your inbox',
            'kind': 'personal',
            'canManage': false,
          },
          {
            'id': 'studio',
            'name': 'Studio',
            'kind': 'team',
            'canManage': false,
          },
        ],
      });
    }
    if (path.endsWith('/items')) {
      return _json({
        'items': [
          {
            'id': path.contains('/studio/') ? 'team' : 'sample',
            'renditionId': 'rendition',
            'title': 'Morning sample',
            'agent': 'Agent',
            'status': 'ready',
            'source': 'Original source',
            'script': 'Spoken briefing',
            'segments': [
              if (shortThenLong)
                {
                  'id': 'short',
                  'kind': 'audio',
                  'durationMs': 2000,
                  'url': 'https://media.example/short.mp3',
                },
              {
                'id': 'audio',
                'kind': 'audio',
                'durationMs': 30000,
                'url': 'https://media.example/sample.mp3',
              },
            ],
          },
        ],
        'nextCursor': null,
      });
    }
    if (path.endsWith('/progress')) {
      final update = request.method == 'PUT'
          ? jsonDecode(request.body) as Map<String, dynamic>
          : <String, dynamic>{};
      return _json({
        'revision': request.method == 'PUT'
            ? (update['expectedRevision'] as int) + 1
            : 0,
        'positionSegment': update['segment'] ?? 0,
        'positionMs': update['positionMs'] ?? 0,
        'segments': [
          if (shortThenLong) {'durationMs': 2000},
          {'durationMs': 30000},
        ],
        'markedRead': update['markedRead'] ?? false,
      });
    }
    if (path.endsWith('/activity')) {
      return _json({'items': [], 'nextCursor': null});
    }
    if (path.endsWith('/settings')) {
      return _json({
        'revision': 0,
        'hasKey': false,
        'notifications': false,
        'textModel': 'text',
        'speechModel': 'speech',
        'voice': 'voice',
      });
    }
    if (path == '/prosody/v1/models') {
      final speech = request.url.queryParameters['kind'] == 'speech';
      return _json({
        'models': [
          {
            'id': speech ? 'speech' : 'text',
            'name': speech ? 'Speech' : 'Text',
            'voices': speech ? ['voice'] : <String>[],
          },
        ],
      });
    }
    throw StateError('Unexpected test route');
  }),
);

Future<void> _flush(WidgetTester tester, Future<void> Function() action) async {
  await tester.runAsync(() async {
    await action();
    await Future<void>.delayed(const Duration(milliseconds: 20));
  });
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'skip clamps to live segment after transition before next frame',
    (tester) async {
      final playback = _Playback(), api = _api(_Session(), shortThenLong: true);
      try {
        await _flush(
          tester,
          () => tester.pumpWidget(
            MaterialApp(
              home: FeedScreen(api: api, playback: playback, onSignOut: () {}),
            ),
          ),
        );
        expect(playback.duration, const Duration(seconds: 2));
        playback.transitionToSegment(1, 1);
        // Do not pump: this taps the previous frame's forward-button callback.
        await tester.tap(find.byTooltip('Forward 10 seconds'));
        expect(playback.seeks.single, const Duration(seconds: 11));
        await tester.pump();
      } finally {
        await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
        api.close();
        await playback.dispose();
      }
    },
  );

  testWidgets('pending directory cannot start playback during sign-out', (
    tester,
  ) async {
    final directory = Completer<http.Response>();
    final session = _Session()..signOutGate = Completer<void>();
    final playback = _Playback(),
        api = _api(session, directory: directory.future);
    var signedOut = 0;
    try {
      await tester.runAsync(() async {
        await tester.pumpWidget(
          MaterialApp(
            home: FeedScreen(
              api: api,
              playback: playback,
              onSignOut: () {
                signedOut++;
              },
            ),
          ),
        );
        await Future<void>.delayed(Duration.zero);
      });
      await tester.pump();
      await tester.runAsync(() async {
        await tester.tap(find.byTooltip('Sign out'));
        await Future<void>.delayed(Duration.zero);
        directory.complete(
          _json({
            'feeds': [
              {
                'id': 'personal',
                'name': 'Your inbox',
                'kind': 'personal',
                'canManage': false,
              },
            ],
          }),
        );
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(session.signOuts, 1);
      expect(signedOut, 0);
      expect(playback.starts, 0);
      expect(playback.current, isNull);
      await _flush(tester, () async {
        session.signOutGate!.complete();
      });
      expect(signedOut, 1);
      expect(playback.starts, 0);
    } finally {
      if (!directory.isCompleted) directory.complete(_json({'feeds': []}));
      if (!session.signOutGate!.isCompleted) session.signOutGate!.complete();
      await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
      api.close();
      await playback.dispose();
    }
  });

  testWidgets(
    'ten-second controls match their icons and clamp at clip boundaries',
    (tester) async {
      final playback = _Playback(), api = _api(_Session());
      try {
        await _flush(
          tester,
          () => tester.pumpWidget(
            MaterialApp(
              home: FeedScreen(api: api, playback: playback, onSignOut: () {}),
            ),
          ),
        );
        for (final (start, tooltip, target) in [
          (14, 'Back 10 seconds', 4),
          (21, 'Back 10 seconds', 11),
          (3, 'Back 10 seconds', 0),
          (25, 'Forward 10 seconds', 30),
          (0, 'Forward 10 seconds', 10),
        ]) {
          playback.setPosition(start);
          await tester.pump();
          await tester.tap(find.byTooltip(tooltip));
          await tester.pump();
          expect(playback.seeks.last, Duration(seconds: target));
        }
        expect(find.byIcon(Icons.replay_10), findsOneWidget);
        expect(find.byIcon(Icons.forward_10), findsOneWidget);
        expect(find.byTooltip('Back 15 seconds'), findsNothing);
      } finally {
        await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
        api.close();
        await playback.dispose();
      }
    },
  );

  testWidgets(
    'minimal header keeps space actions in selector and signs out directly',
    (tester) async {
      final session = _Session();
      final playback = _Playback(), api = _api(session);
      var signedOut = 0;
      try {
        await _flush(
          tester,
          () => tester.pumpWidget(
            MaterialApp(
              home: FeedScreen(
                api: api,
                playback: playback,
                onSignOut: () {
                  signedOut++;
                },
              ),
            ),
          ),
        );
        final header = find.byType(AppBar);
        expect(
          find.descendant(of: header, matching: find.byType(ProsodyMark)),
          findsNothing,
        );
        expect(
          find.descendant(of: header, matching: find.byType(IconButton)),
          findsOneWidget,
        );
        expect(
          tester.getCenter(find.text('Your inbox')).dx,
          lessThan(tester.getCenter(find.byIcon(Icons.expand_more)).dx),
        );
        expect(
          tester.getCenter(find.byIcon(Icons.expand_more)).dx,
          lessThan(tester.getCenter(find.byTooltip('Sign out')).dx),
        );
        await _flush(tester, () => tester.tap(find.text('Your inbox')));
        expect(find.text('Briefing activity'), findsOneWidget);
        expect(find.text('Space settings'), findsOneWidget);
        await _flush(tester, () => tester.tap(find.text('Briefing activity')));
        expect(find.byType(ActivityScreen), findsOneWidget);
        await _flush(tester, () => tester.pageBack());
        await _flush(tester, () => tester.tap(find.text('Your inbox')));
        await tester.ensureVisible(find.text('Space settings'));
        await tester.pumpAndSettle();
        await _flush(tester, () => tester.tap(find.text('Space settings')));
        expect(find.byType(SettingsScreen), findsOneWidget);
        await _flush(tester, () => tester.pageBack());
        await _flush(tester, () => tester.tap(find.text('Your inbox')));
        await _flush(tester, () => tester.tap(find.text('Studio')));
        expect(playback.current?.id, 'team');
        final beforeStop = playback.stops;
        session.signOutGate = Completer<void>();
        await tester.runAsync(() async {
          await tester.tap(find.byTooltip('Sign out'));
          await Future<void>.delayed(Duration.zero);
        });
        await tester.pump();
        expect(
          tester
              .widget<IconButton>(
                find.widgetWithIcon(IconButton, Icons.logout_rounded),
              )
              .onPressed,
          isNull,
        );
        expect(playback.stops, beforeStop + 1);
        expect(session.signOuts, 1);
        expect(signedOut, 0);
        await _flush(tester, () async {
          session.signOutGate!.complete();
        });
        expect(signedOut, 1);
      } finally {
        if (session.signOutGate != null && !session.signOutGate!.isCompleted) {
          session.signOutGate!.complete();
        }
        await _flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
        api.close();
        await playback.dispose();
      }
    },
  );
}
