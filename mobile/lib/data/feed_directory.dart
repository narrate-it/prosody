import 'models.dart';
import 'pagination.dart';

// Bounds an explicit directory refresh. Every page still requires server-side
// current authority; cursors and canManage flags do not authorize effects.
Future<List<Feed>> loadFeedDirectory(
  Future<Map<String, dynamic>> Function(String? cursor) fetch, {
  int maxPages = 20,
  int maxFeeds = 500,
}) async {
  if (maxPages < 1 || maxFeeds < 1) {
    throw ArgumentError('Positive directory limits required');
  }
  final pagination = FeedPagination();
  final feeds = <Feed>[];
  final ids = <String>{};
  String? cursor;
  for (var page = 0; page < maxPages; page++) {
    final value = await fetch(cursor);
    final entries = value['feeds'], next = value['nextCursor'];
    if (entries is! List || (next != null && next is! String)) {
      throw const FormatException('Invalid feed directory');
    }
    pagination.accept(requested: cursor, next: next as String?);
    for (final entry in entries) {
      if (entry is! Map<String, dynamic> ||
          entry['id'] is! String ||
          (entry['id'] as String).isEmpty ||
          entry['name'] is! String ||
          !const ['personal', 'team'].contains(entry['kind']) ||
          entry['canManage'] is! bool) {
        throw const FormatException('Invalid feed directory entry');
      }
      final feed = Feed.fromJson(entry);
      if (!ids.add(feed.id)) {
        throw const FormatException('Duplicate feed identity');
      }
      if (feeds.length == maxFeeds) {
        throw const FormatException('Feed directory exceeds display limit');
      }
      feeds.add(feed);
    }
    cursor = pagination.next;
    if (cursor == null) {
      return List.unmodifiable(feeds);
    }
  }
  throw const FormatException('Feed directory exceeds page limit');
}
