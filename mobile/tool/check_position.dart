import 'dart:io';
import 'package:prosody/playback/position.dart';
import 'package:prosody/playback/coverage.dart';

void checkPosition() {
  final position = ResumePosition();
  final coverage = CoverageSampler();
  position.observe(5000, durationMs: 10000);
  coverage.sample(
    positionMs: 5000,
    monotonicMs: 100,
    playing: false,
    ready: true,
  );
  if (!position.dirty || position.value != 5000 || coverage.take().isNotEmpty) {
    throw StateError(
      'Paused seek failed to separate position from consumption',
    );
  }
  position.captured();
  position.observe(5000, durationMs: 10000);
  if (position.dirty) throw StateError('Unchanged position repeatedly dirtied');
  position.observe(1000, durationMs: 10000);
  if (!position.dirty || position.value != 1000)
    throw StateError('Backward seek lost');
  position.observe(20000, durationMs: 10000);
  if (position.value != 10000) throw StateError('Decoder overshoot persisted');
  position.observe(-10, durationMs: 10000);
  if (position.value != 0) throw StateError('Negative position persisted');
  position.reset();
  if (position.dirty || position.value != 0)
    throw StateError('New segment inherited position');
  position.observe(5000, durationMs: 0);
  if (position.dirty || position.value != 0)
    throw StateError('Image inherited audio position');
}

void main() {
  checkPosition();
  stdout.writeln(
    'Resume position checks passed; recorder/device integration untested.',
  );
}
