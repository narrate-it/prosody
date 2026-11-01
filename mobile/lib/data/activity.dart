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
    final timestamp = DateTime.tryParse(accepted);
    if (timestamp == null || !timestamp.isUtc) {
      throw const FormatException('Invalid activity timestamp');
    }
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
