import 'dart:convert';
import 'package:http/http.dart' as http;
import 'models.dart';
import 'activity.dart';
import 'bounded_body.dart';
import 'feed_directory.dart';
import 'mark_read.dart';

class ApiFailure implements Exception {
  final int status;
  final String message;
  const ApiFailure(this.status, this.message);
  @override
  String toString() => message;
}

abstract interface class AccessSession {
  Future<String> accessToken();
  Future<void> signOut();
}

class ProsodyApi {
  final Uri origin;
  final AccessSession session;
  final http.Client _client;
  ProsodyApi(this.origin, this.session, {http.Client? client})
    : _client = client ?? http.Client() {
    if (origin.scheme != 'https' ||
        origin.host.isEmpty ||
        origin.userInfo.isNotEmpty ||
        origin.hasQuery ||
        origin.hasFragment ||
        (origin.path.isNotEmpty && origin.path != '/')) {
      throw ArgumentError('Service origin must be HTTPS');
    }
  }
  Future<Map<String, dynamic>> request(
    String method,
    String path, {
    Map<String, String>? query,
    Object? body,
  }) async {
    if (!path.startsWith('/prosody/v1/')) {
      throw ArgumentError('Invalid API route');
    }
    final token = await session.accessToken();
    final request = http.Request(
      method,
      origin.replace(path: path, queryParameters: query),
    );
    request.followRedirects = false;
    request.headers.addAll({
      'Authorization': 'Bearer $token',
      'Accept': 'application/json',
    });
    if (body != null) {
      request.headers['Content-Type'] = 'application/json';
      request.body = jsonEncode(body);
    }
    final response = await _client
        .send(request)
        .timeout(const Duration(seconds: 30));
    final List<int> bytes;
    try {
      bytes = await collectBoundedBody(
        response.stream,
        maxBytes: 8 * 1024 * 1024,
        deadline: const Duration(seconds: 30),
      );
    } on BodyTooLarge {
      throw const ApiFailure(0, 'Response too large.');
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiFailure(response.statusCode, switch (response.statusCode) {
        401 => 'Your session has expired. Sign in again.',
        403 => 'You no longer have access to this space.',
        409 => 'This changed on another device. Refresh and try again.',
        429 => 'Please wait a moment and try again.',
        _ => 'Could not complete this request. Please try again.',
      });
    }
    return bytes.isEmpty
        ? <String, dynamic>{}
        : objectFromJson(utf8.decode(bytes));
  }

  Future<List<Feed>> feeds() => loadFeedDirectory(
    (cursor) => request(
      'GET',
      '/prosody/v1/feeds',
      query: cursor == null ? null : {'cursor': cursor},
    ),
  );
  Future<FeedPage> items(String feed, {String? cursor}) async =>
      FeedPage.fromJson(
        await request(
          'GET',
          '/prosody/v1/feeds/${Uri.encodeComponent(feed)}/items',
          query: cursor == null ? null : {'cursor': cursor},
        ),
      );
  // Proposed consumption route; production composition awaits AMOS authority.
  Future<ActivityPage> activity(String feed, {String? cursor}) async =>
      ActivityPage.fromJson(
        await request(
          'GET',
          '/prosody/v1/feeds/${Uri.encodeComponent(feed)}/activity',
          query: cursor == null ? null : {'cursor': cursor},
        ),
      );
  Future<FeedSettings> settings(String feed) async => FeedSettings.fromJson(
    await request(
      'GET',
      '/prosody/v1/feeds/${Uri.encodeComponent(feed)}/settings',
    ),
  );
  Future<List<ProviderModel>> models(String kind) async =>
      providerCatalogFromJson(
        (await request(
          'GET',
          '/prosody/v1/models',
          query: {'kind': kind},
        ))['models'],
      );
  Future<void> saveSettings(String feed, Map<String, dynamic> body) async {
    await request(
      'PUT',
      '/prosody/v1/feeds/${Uri.encodeComponent(feed)}/settings',
      body: body,
    );
  }

  Future<void> notifications(String feed, bool enabled) async {
    await request(
      'PUT',
      '/prosody/v1/feeds/${Uri.encodeComponent(feed)}/notifications',
      body: {'enabled': enabled},
    );
  }

  Future<Map<String, dynamic>> progress(String item, String rendition) =>
      request(
        'GET',
        '/prosody/v1/items/${Uri.encodeComponent(item)}/progress',
        query: {'renditionId': rendition},
      );
  Future<Map<String, dynamic>> updateProgress(
    String item,
    Map<String, dynamic> update,
  ) => request(
    'PUT',
    '/prosody/v1/items/${Uri.encodeComponent(item)}/progress',
    body: update,
  );
  // No automatic retry: a lost response may follow a committed mutation.
  Future<void> markRead(Briefing item) async {
    final saved = SavedProgress.fromJson(
      await progress(item.id, item.renditionId),
      item,
    );
    final result = await updateProgress(item.id, markReadUpdate(item, saved));
    SavedProgress.fromJson(
      result,
      item,
      expectedPreviousRevision: saved.revision,
    );
    if (result['markedRead'] != true) {
      throw const FormatException('Read status was not confirmed');
    }
  }

  Future<void> deleteItem(String item) async {
    await request('DELETE', '/prosody/v1/items/${Uri.encodeComponent(item)}');
  }

  void close() => _client.close();
}
