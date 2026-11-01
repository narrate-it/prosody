import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'directory-test-session';
  @override
  Future<void> signOut() async {}
}

Future<void> checkDirectoryTransport() async {
  const cursor = 'opaque+/= cursor';
  for (final mode in ['success', 'revoked', 'cycle', 'duplicate']) {
    var calls = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        calls++;
        if (request.method != 'GET' ||
            request.url.path != '/prosody/v1/feeds' ||
            request.headers['Authorization'] !=
                'Bearer directory-test-session' ||
            request.followRedirects ||
            request.url.queryParameters['cursor'] !=
                (calls == 1 ? null : cursor)) {
          throw StateError('Incorrect directory request');
        }
        if (calls > 2) {
          throw StateError('Unexpected retry');
        }
        if (calls == 2 && mode == 'revoked') {
          return http.Response('private authority details', 403);
        }
        return http.Response(
          jsonEncode({
            'feeds': [
              {
                'id': calls == 1 || mode == 'duplicate' ? 'personal' : 'team',
                'name': calls == 1 ? 'My inbox' : 'Team',
                'kind': calls == 1 ? 'personal' : 'team',
                'canManage': calls == 1,
              },
            ],
            'nextCursor': calls == 1 || mode == 'cycle' ? cursor : null,
          }),
          200,
        );
      }),
    );
    Object? failure;
    try {
      final feeds = await api.feeds();
      if (mode != 'success' ||
          feeds.length != 2 ||
          feeds.last.id != 'team' ||
          feeds.last.canManage ||
          !feeds.first.canManage) {
        throw StateError('Partial or incorrect directory returned');
      }
    } catch (error) {
      failure = error;
    } finally {
      api.close();
    }
    final valid = switch (mode) {
      'success' => failure == null,
      'revoked' =>
        failure is ApiFailure &&
            failure.status == 403 &&
            !failure.message.contains('private'),
      _ => failure is FormatException,
    };
    if (!valid || calls != 2) {
      throw StateError('Unexpected directory outcome: $mode, $failure');
    }
  }

  for (final mode in ['success', 'invalid', 'revoked']) {
    var calls = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        calls++;
        if (request.method != 'GET' ||
            request.url.path != '/prosody/v1/feeds/team/activity' ||
            request.url.queryParameters['cursor'] != cursor ||
            request.headers['Authorization'] !=
                'Bearer directory-test-session' ||
            request.followRedirects) {
          throw StateError('Incorrect activity request');
        }
        if (mode == 'revoked') {
          return http.Response('private authority details', 403);
        }
        return http.Response(
          jsonEncode({
            'items': [
              {
                'id': 'submission',
                'title': 'Update',
                'source': 'Text\u0000retained',
                'status': mode == 'invalid' ? 'unknown' : 'ready',
                'acceptedAt': '2026-11-01T12:00:00Z',
              },
            ],
            'nextCursor': null,
          }),
          200,
        );
      }),
    );
    Object? failure;
    try {
      final page = await api.activity('team', cursor: cursor);
      if (page.items.single.source != 'Text\u0000retained' ||
          page.items.single.status != 'ready' ||
          page.nextCursor != null) {
        throw StateError('Activity summary altered');
      }
    } catch (error) {
      failure = error;
    } finally {
      api.close();
    }
    final valid = switch (mode) {
      'success' => failure == null,
      'invalid' => failure is FormatException,
      _ =>
        failure is ApiFailure &&
            failure.status == 403 &&
            !failure.message.contains('private'),
    };
    if (!valid || calls != 1) {
      throw StateError('Unexpected activity outcome: $mode, $failure');
    }
  }
}

Future<void> main() async {
  await checkDirectoryTransport();
  stdout.writeln(
    'Directory and Activity API mock checks passed; no real service exercised.',
  );
}
