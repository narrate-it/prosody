import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-only-session';
  @override
  Future<void> signOut() async {}
}

Future<void> checkMarkReadTransport() async {
  final item = Briefing.fromJson({
    'id': 'item',
    'renditionId': 'rendition',
    'title': 'Title',
    'agent': 'Agent',
    'status': 'ready',
    'segments': [
      {
        'id': 'audio',
        'kind': 'audio',
        'durationMs': 1000,
        'url': 'https://media.example/audio',
      },
    ],
  });
  final progress = <String, dynamic>{
    'revision': 8,
    'positionSegment': 0,
    'positionMs': 400,
    'segments': [
      {'durationMs': 1000},
    ],
    'markedRead': false,
  };
  for (final mode in [
    'success',
    'denied',
    'bad-read',
    'conflict',
    'lost-response',
    'unconfirmed',
    'stale-result',
  ]) {
    var reads = 0, writes = 0;
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      _Session(),
      client: MockClient((request) async {
        if (request.url.path != '/prosody/v1/items/item/progress' ||
            request.headers['Authorization'] != 'Bearer test-only-session' ||
            request.followRedirects) {
          throw StateError('Unexpected authenticated route');
        }
        if (request.method == 'GET') {
          reads++;
          if (request.url.queryParameters['renditionId'] != 'rendition') {
            throw StateError('Rendition selector lost');
          }
          if (mode == 'denied') {
            return http.Response('private error detail', 403);
          }
          return http.Response(
            jsonEncode({
              ...progress,
              if (mode == 'bad-read') 'positionMs': 2000,
            }),
            200,
          );
        }
        if (request.method != 'PUT') throw StateError('Unexpected mutation');
        writes++;
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        if (body['expectedRevision'] != 8 ||
            body['renditionId'] != 'rendition' ||
            body['positionMs'] != 400 ||
            body['segment'] != 0 ||
            body['markedRead'] != true ||
            body['viewed'] != false ||
            (body['played'] as List).isNotEmpty) {
          throw StateError('Mark-read changed saved position or consumption');
        }
        if (mode == 'conflict') {
          return http.Response('private conflict detail', 409);
        }
        if (mode == 'lost-response') {
          throw TimeoutException('test response lost after commit');
        }
        return http.Response(
          jsonEncode({
            ...progress,
            'revision': mode == 'stale-result' ? 8 : 9,
            'markedRead': mode != 'unconfirmed',
          }),
          200,
        );
      }),
    );
    Object? failure;
    try {
      await api.markRead(item);
    } catch (error) {
      failure = error;
    } finally {
      api.close();
    }
    if (reads != 1 ||
        writes != (mode == 'denied' || mode == 'bad-read' ? 0 : 1)) {
      throw StateError('Unexpected retry or unauthorized mutation: $mode');
    }
    final valid = switch (mode) {
      'success' => failure == null,
      'denied' => failure is ApiFailure && failure.status == 403,
      'conflict' => failure is ApiFailure && failure.status == 409,
      'lost-response' => failure is TimeoutException,
      _ => failure is FormatException,
    };
    if (!valid) throw StateError('Incorrect outcome for $mode');
    if (failure is ApiFailure && failure.message.contains('private')) {
      throw StateError('Private response detail exposed');
    }
  }
}

Future<void> main() async {
  await checkMarkReadTransport();
  stdout.writeln(
    'Mark-read API mock checks passed; no real service exercised.',
  );
}
