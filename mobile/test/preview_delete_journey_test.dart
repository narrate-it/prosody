import 'dart:async';
import 'dart:io';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:prosody/data/models.dart';
import 'package:prosody/playback/handler.dart';
import 'package:prosody/preview/local_media.dart';
import 'package:prosody/preview/live_session.dart';
import 'package:prosody/preview/preview_api.dart';
import 'package:prosody/preview_main.dart';

class _Audio extends Fake implements audio.AudioPlayer {}

class _Player extends ProsodyPlayback {
  _Player() : super(audioPlayer: _Audio());
  Briefing? _current;
  final List<String> stopped = [];
  final List<List<String>> queues = [];
  @override
  Briefing? get current => _current;
  @override
  Duration get position => Duration.zero;
  @override
  Duration get duration => _current?.duration ?? Duration.zero;
  @override
  Future<void> stop() async {
    if (_current != null) stopped.add(_current!.id);
    _current = null;
    location.value = const PlaybackLocation(0, 0);
  }

  @override
  Future<void> replaceFeed(
    List<Briefing> items, {
    int initial = 0,
    int initialSegment = 0,
    Duration initialPosition = Duration.zero,
  }) async {
    queues.add(items.map((item) => item.id).toList());
    _current = items.isEmpty ? null : items[initial];
    location.value = PlaybackLocation(initial, initialSegment);
    playbackState.add(
      PlaybackState(
        processingState: AudioProcessingState.ready,
        playing: false,
        updatePosition: initialPosition,
      ),
    );
  }

  @override
  void appendItems(List<Briefing> items) {}
  @override
  Future<void> dispose() => stop();
}

Future<void> flush(WidgetTester tester, Future<void> Function() action) async {
  await tester.runAsync(() async {
    await action();
    await Future<void>.delayed(const Duration(milliseconds: 20));
  });
  await tester.pumpAndSettle();
}

void main() {
  for (final failFirst in [false, true]) {
    testWidgets(
      'personal preview delete refreshes real API queue; initial failure=$failFirst',
      (tester) async {
        final directory = Directory.systemTemp.createTempSync(
          'prosody-delete-journey-',
        );
        final clip = File('${directory.path}/live-123.mp3')
          ..writeAsBytesSync([1, 2, 3]);
        final ledger = File('${directory.path}/budget');
        final player = _Player();
        final files = {'https://preview.invalid/live-123.mp3': clip.path};
        final live = LiveSession(ledger: ledger);
        var attempts = 0;
        late Completer<void> cleanup;
        final api = PreviewApi(
          durationsMs: {'welcome': 12000, 'design': 15000, 'next': 13000},
          onDelete: (id) async {
            try {
              attempts++;
              expect(player.current, isNull);
              expect(player.stopped, contains('live-123'));
              if (failFirst && attempts == 1) {
                throw FileSystemException('synthetic failure');
              }
              await deletePreviewMedia(files, id);
            } finally {
              cleanup.complete();
            }
          },
        );
        api.addLiveBriefing(
          id: 'live-123',
          source: 'Source',
          script: 'Briefing',
          durationMs: 20000,
        );
        try {
          await tester.runAsync(() async {
            await live.initialize();
          });
          final budgetBefore = ledger.readAsStringSync();
          await flush(
            tester,
            () => tester.pumpWidget(
              ProsodyPreview(
                playback: player,
                api: api,
                live: live,
                clips: directory,
                files: files,
              ),
            ),
          );
          expect(player.current?.id, 'live-123');
          await flush(tester, () => tester.tap(find.text('Read')));
          final rounds = failFirst ? 2 : 1;
          for (var round = 1; round <= rounds; round++) {
            await tester.runAsync(() async {
              cleanup = Completer<void>();
            });
            await flush(tester, () => tester.tap(find.text('Delete briefing')));
            await flush(tester, () async {
              await tester.tap(find.text('Delete'));
              await cleanup.future.timeout(const Duration(seconds: 5));
            });
            expect(attempts, round);
            if (round < rounds) {
              expect(
                find.text('Could not delete this briefing. Try again.'),
                findsOneWidget,
              );
              expect(clip.existsSync(), isTrue);
              expect(player.current?.id, 'live-123');
              expect(player.queues.last.first, 'live-123');
            }
          }
          expect(find.text('Delete briefing'), findsNothing);
          expect(clip.existsSync(), isFalse);
          expect(files, isEmpty);
          expect(player.current?.id, 'welcome');
          expect(player.queues.last, ['welcome', 'next']);
          expect(ledger.readAsStringSync(), budgetBefore);
          expect(tester.takeException(), isNull);
        } finally {
          await flush(tester, () => tester.pumpWidget(const SizedBox.shrink()));
          directory.deleteSync(recursive: true);
        }
      },
    );
  }
}
