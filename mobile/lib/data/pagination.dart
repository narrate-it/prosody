// Tracks only accepted pages. Failed requests may retry the current cursor.
// A fresh instance belongs to each feed load; stale responses must be discarded
// by the caller before acceptance.
class FeedPagination {
  final Set<String> _seen = {};
  String? _next;
  bool _started = false;
  String? get next => _next;

  void accept({required String? requested, required String? next}) {
    if ((!_started && requested != null) ||
        (_started && (_next == null || requested != _next)) ||
        (next != null && (next.isEmpty || _seen.contains(next)))) {
      throw const FormatException('Invalid feed pagination sequence');
    }
    if (next != null) _seen.add(next);
    _next = next;
    _started = true;
  }
}
