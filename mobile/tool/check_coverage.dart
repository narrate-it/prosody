import 'dart:io';
// Dependency-free arithmetic checks; no platform playback is exercised.
import 'package:prosody/playback/coverage.dart';

void checkCoverage() {
  void expectIntervals(CoverageSampler s, List<List<int>> expected) {
    final actual = s.take().map((v) => [v.startMs, v.endMs]).toList();
    if (actual.toString() != expected.toString()) {
      throw StateError('Expected $expected, received $actual');
    }
  }

  void sample(
    CoverageSampler s,
    int position,
    int wall, {
    bool playing = true,
    bool ready = true,
    double speed = 1,
  }) {
    s.sample(
      positionMs: position,
      monotonicMs: wall,
      playing: playing,
      ready: ready,
      speed: speed,
    );
  }

  final continuous = CoverageSampler();
  sample(continuous, 0, 0);
  sample(continuous, 100, 100);
  sample(continuous, 200, 200);
  expectIntervals(continuous, [
    [0, 200],
  ]);
  expectIntervals(continuous, []);
  sample(continuous, 300, 300);
  expectIntervals(continuous, [
    [200, 300],
  ]);

  final seek = CoverageSampler();
  sample(seek, 0, 0);
  sample(seek, 1000, 1000);
  seek.discontinuity();
  sample(seek, 50000, 1100);
  sample(seek, 51000, 2100);
  sample(seek, 60000, 11100);
  expectIntervals(seek, [
    [0, 1000],
    [50000, 51000],
  ]);

  for (final buffering in [false, true]) {
    final paused = CoverageSampler();
    sample(paused, 0, 0);
    sample(paused, 1000, 1000, playing: buffering, ready: !buffering);
    sample(paused, 2000, 2000);
    sample(paused, 2100, 2100);
    expectIntervals(paused, [
      [2000, 2100],
    ]);
  }
  for (final speed in [double.nan, double.infinity, -1.0, 0.0]) {
    final invalid = CoverageSampler();
    sample(invalid, 0, 0);
    sample(invalid, 10000, 100, speed: speed);
    sample(invalid, 10100, 200);
    expectIntervals(invalid, []);
  }
  for (final invalidPosition in [true, false]) {
    final invalid = CoverageSampler();
    sample(invalid, invalidPosition ? -100 : 0, invalidPosition ? 0 : -100);
    sample(invalid, 100, 100);
    expectIntervals(invalid, []);
  }
  final fast = CoverageSampler();
  sample(fast, 0, 0, speed: 2);
  sample(fast, 1000, 500, speed: 2);
  expectIntervals(fast, [
    [0, 1000],
  ]);

  final backwards = CoverageSampler();
  sample(backwards, 1000, 1000);
  sample(backwards, 0, 1100);
  sample(backwards, 100, 1200);
  expectIntervals(backwards, [
    [0, 100],
  ]);

  final shortClip = CoverageSampler();
  sample(shortClip, 0, 0);
  sample(shortClip, 80, 80);
  expectIntervals(shortClip, [
    [0, 80],
  ]);
}

void main() {
  checkCoverage();
  stdout.writeln(
    'Coverage arithmetic checks passed (no platform playback exercised).',
  );
}
