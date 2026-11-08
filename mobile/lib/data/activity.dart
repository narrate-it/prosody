// Submission summaries are not playback manifests. In particular, a ready
// summary does not grant asset access or require rendition/segment data.
class ActivitySummary {
  final String id, title, source, status;
  final DateTime acceptedAt;
  const ActivitySummary({
    required this.id,
    required this.title,
    required this.source,
    required this.status,
    required this.acceptedAt,
  });

  factory ActivitySummary.fromJson(Map<String, dynamic> value) {
    final id = value['id'], title = value['title'], source = value['source'];
    final status = value['status'], accepted = value['acceptedAt'];
    if (id is! String ||
        id.isEmpty ||
        title is! String ||
        source is! String ||
        !const ['processing', 'failed', 'ready'].contains(status) ||
        accepted is! String) {
      throw const FormatException('Invalid activity summary');
    }
    final timestamp = _parseActivityTimestamp(accepted);
    return ActivitySummary(
      id: id,
      title: title,
      source: source,
      status: status as String,
      acceptedAt: timestamp,
    );
  }
}

class ActivityPage {
  final List<ActivitySummary> items;
  final String? nextCursor;
  const ActivityPage(this.items, this.nextCursor);
  factory ActivityPage.fromJson(Map<String, dynamic> value) {
    final items = value['items'], cursor = value['nextCursor'];
    if (items is! List ||
        (cursor != null && (cursor is! String || cursor.isEmpty))) {
      throw const FormatException('Invalid activity page');
    }
    final summaries = <ActivitySummary>[];
    for (final item in items) {
      if (item is! Map<String, dynamic>) {
        throw const FormatException('Invalid activity entry');
      }
      summaries.add(ActivitySummary.fromJson(item));
    }
    return ActivityPage(List.unmodifiable(summaries), cursor as String?);
  }
}

final _activityTimestampPattern = RegExp(
  r'([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.[0-9]+)?(?:Z|([+-])([0-9]{2}):([0-9]{2}))',
);

/// Decode the server's RFC 3339 timestamp without normalizing invalid fields.
///
/// DateTime.tryParse alone accepts February 30 and overflowing time/offset fields.
/// Validate the original calendar and clock before applying its explicit offset.
DateTime _parseActivityTimestamp(String value) {
  final match = _activityTimestampPattern.matchAsPrefix(value);
  if (match == null || match.end != value.length) {
    throw const FormatException('Invalid activity timestamp');
  }
  final fields = [for (var i = 1; i <= 6; i++) int.parse(match.group(i)!)];
  final calendar = DateTime.utc(
    fields[0],
    fields[1],
    fields[2],
    fields[3],
    fields[4],
    fields[5],
  );
  if (calendar.year != fields[0] ||
      calendar.month != fields[1] ||
      calendar.day != fields[2] ||
      calendar.hour != fields[3] ||
      calendar.minute != fields[4] ||
      calendar.second != fields[5] ||
      (match.group(8) != null && int.parse(match.group(8)!) > 23) ||
      (match.group(9) != null && int.parse(match.group(9)!) > 59)) {
    throw const FormatException('Invalid activity timestamp');
  }
  final timestamp = DateTime.tryParse(value);
  if (timestamp == null || !timestamp.isUtc) {
    throw const FormatException('Invalid activity timestamp');
  }
  return timestamp;
}
