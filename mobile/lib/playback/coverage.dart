// Played intervals come from continuous decoder progress, never seek distance.
class PlayedInterval {
  final int startMs, endMs;
  const PlayedInterval(this.startMs, this.endMs);
  Map<String, int> toJson() => {'startMs': startMs, 'endMs': endMs};
}

class CoverageSampler {
  int? _position, _wall;
  final List<PlayedInterval> _intervals = [];
  void discontinuity() {
    _position = null;
    _wall = null;
  }

  void sample({
    required int positionMs,
    required int monotonicMs,
    required bool playing,
    required bool ready,
    double speed = 1,
  }) {
    final previous = _position, wall = _wall;
    _position = positionMs;
    _wall = monotonicMs;
    if (!playing ||
        !ready ||
        positionMs < 0 ||
        monotonicMs < 0 ||
        !speed.isFinite ||
        speed <= 0) {
      discontinuity();
      return;
    }
    if (previous == null || wall == null) return;
    final elapsed = monotonicMs - wall, advanced = positionMs - previous;
    // Gaps (background scheduling delays/buffering) are undercounted rather
    // than invented. Seek must additionally call discontinuity explicitly.
    if (elapsed <= 0 ||
        elapsed > 2500 ||
        advanced <= 0 ||
        advanced > elapsed * speed + 200) {
      return;
    }
    if (_intervals.isNotEmpty && _intervals.last.endMs == previous) {
      final last = _intervals.removeLast();
      _intervals.add(PlayedInterval(last.startMs, positionMs));
    } else {
      _intervals.add(PlayedInterval(previous, positionMs));
    }
  }

  List<PlayedInterval> take() {
    final result = List<PlayedInterval>.of(_intervals);
    _intervals.clear();
    return result;
  }
}
