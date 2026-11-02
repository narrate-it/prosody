import 'dart:async';
import 'package:flutter_test/flutter_test.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:prosody/data/models.dart';
import 'package:prosody/playback/handler.dart';

class _Audio extends Fake implements audio.AudioPlayer {
  final stops = <Completer<void>>[];
  bool delayStops = false;
  @override
  Future<void> stop() {
    if (!delayStops) return Future.value();
    final result = Completer<void>();
    stops.add(result);
    return result.future;
  }

  @override
  Future<void> pause() async {}
  @override
  Future<void> dispose() async {}
  @override
  double get speed => 1;
  @override
  Duration get position => Duration.zero;
  @override
  Duration get bufferedPosition => Duration.zero;
  @override
  audio.ProcessingState get processingState => audio.ProcessingState.idle;
}

Briefing _visual(String id) => Briefing(
  id: id,
  renditionId: '$id-r',
  title: id,
  agent: 'Agent',
  status: 'ready',
  source: '',
  script: '',
  segments: [
    Segment(
      id: 'image',
      kind: 'image',
      duration: Duration.zero,
      images: [Uri.parse('https://example.test/image')],
    ),
  ],
);

// Use the real event loop: audio_service's asynchronous stream delivery is part
// of the handler boundary exercised here. Bound each operation independently.
Future<void> _finish(Future<void> operation) =>
    operation.timeout(const Duration(seconds: 3));

void main() {
  test(
    'visual dwell advances without crediting a view; pause cancels it',
    () async {
      final handler = ProsodyPlayback(audioPlayer: _Audio());
      await _finish(handler.replaceFeed([_visual('first'), _visual('second')]));
      expect(handler.current?.id, 'first');
      await Future<void>.delayed(const Duration(milliseconds: 5100));
      expect(handler.current?.id, 'second');
      expect(handler.visibleImages, isEmpty);
      expect(handler.completed.value, 0);
      await _finish(handler.pause());
      await Future<void>.delayed(const Duration(seconds: 6));
      expect(handler.current?.id, 'second');
      expect(handler.isPlaying, isFalse);
      await _finish(handler.play());
      await _finish(handler.stop());
      await Future<void>.delayed(const Duration(seconds: 6));
      expect(handler.current, isNull);
      expect(handler.isPlaying, isFalse);
      await _finish(handler.dispose());
    },
    timeout: const Timeout(Duration(seconds: 45)),
  );

  test('a stopped stale replacement cannot resurrect its queue', () async {
    final audioPlayer = _Audio()..delayStops = true;
    final handler = ProsodyPlayback(audioPlayer: audioPlayer);
    final replacement = handler.replaceFeed([_visual('stale')]);
    final stopping = handler.stop();
    expect(audioPlayer.stops, hasLength(2));
    // Simulate the newer stop finishing first (e.g. sign-out).
    audioPlayer.stops[1].complete();
    await _finish(stopping);
    audioPlayer.stops[0].complete();
    await _finish(replacement);
    expect(handler.current, isNull);
    expect(handler.queue.value, isEmpty);
    expect(handler.isPlaying, isFalse);
    audioPlayer.delayStops = false;
    await _finish(handler.dispose());
  });
}
