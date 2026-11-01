import 'dart:io';
import 'package:prosody/playback/timeline.dart';

void checkTimeline() {
  void expect(List<int> durations, int requested, int segment, int position) {
    final actual = seekInTimeline(
      durations.map((ms) => Duration(milliseconds: ms)).toList(),
      Duration(milliseconds: requested),
    );
    if (actual.segment != segment ||
        actual.position.inMilliseconds != position) {
      throw StateError('Incorrect mixed-media seek destination');
    }
  }

  expect([0, 1000, 0, 2000, 0], -100, 1, 0);
  expect([0, 1000, 0, 2000, 0], 0, 1, 0);
  expect([0, 1000, 0, 2000, 0], 999, 1, 999);
  expect([0, 1000, 0, 2000, 0], 1000, 3, 0);
  expect([0, 1000, 0, 2000, 0], 2500, 3, 1500);
  expect([0, 1000, 0, 2000, 0], 3000, 3, 2000);
  expect([0, 1000, 0, 2000, 0], 99999, 3, 2000);
  expect([0, 0], 1000, 0, 0);
  expect([1000], 1000, 0, 1000);
  for (final durations in <List<Duration>>[
    [],
    [const Duration(seconds: -1)],
  ]) {
    var rejected = false;
    try {
      seekInTimeline(durations, Duration.zero);
    } on FormatException {
      rejected = true;
    }
    if (!rejected) throw StateError('Invalid timeline accepted');
  }
}

void main() {
  checkTimeline();
  stdout.writeln('Mixed-media seek checks passed; no player/device exercised.');
}
