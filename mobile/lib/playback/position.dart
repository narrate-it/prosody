// Resume position and consumption are independent. A seek can dirty the former
// without creating played intervals or marking anything viewed/read.
class ResumePosition {
  int _value = 0;
  bool _dirty = false;
  int get value => _value;
  bool get dirty => _dirty;

  void observe(int milliseconds, {required int durationMs}) {
    if (durationMs < 0) {
      throw ArgumentError('Negative segment duration');
    }
    final bounded = milliseconds.clamp(0, durationMs);
    if (bounded != _value) {
      _value = bounded;
      _dirty = true;
    }
  }

  void captured() {
    _dirty = false;
  }

  void reset() {
    _value = 0;
    _dirty = false;
  }
}
