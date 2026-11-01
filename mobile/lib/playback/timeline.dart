// OS seek positions describe timed media. Images have no duration and are
// visited by normal segment progression, never credited as viewed by a seek.
class TimelinePosition {
  final int segment;
  final Duration position;
  const TimelinePosition(this.segment, this.position);
}

TimelinePosition seekInTimeline(List<Duration> durations, Duration requested) {
  if (durations.isEmpty || durations.any((d) => d.isNegative)) {
    throw const FormatException('Invalid playback timeline');
  }
  var remaining = requested.isNegative ? Duration.zero : requested;
  int? lastTimed;
  for (var index = 0; index < durations.length; index++) {
    final duration = durations[index];
    if (duration == Duration.zero) {
      continue;
    }
    lastTimed = index;
    if (remaining < duration) {
      return TimelinePosition(index, remaining);
    }
    remaining -= duration;
  }
  if (lastTimed == null) {
    return const TimelinePosition(0, Duration.zero);
  }
  return TimelinePosition(lastTimed, durations[lastTimed]);
}
