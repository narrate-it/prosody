import 'dart:async';
import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/data/resume.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-only';
  @override
  Future<void> signOut() async {}
}

void main() {
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
  for (final status in [401, 403]) {
    for (final bodyMode in [
      'failing',
      'oversized',
      'never-ending',
      'cancel-fails',
    ]) {
      test('known $status survives $bodyMode streamed body', () async {
        var calls = 0, cancellations = 0;
        final cancellationGate = Completer<void>();
        late StreamController<List<int>> body;
        body = StreamController<List<int>>(
          onListen: () {
            if (bodyMode == 'failing') {
              body.addError(http.ClientException('private body failure'));
            } else if (bodyMode == 'oversized') {
              body.add(List<int>.filled(8 * 1024 * 1024 + 1, 65));
            }
          },
          onCancel: () {
            cancellations++;
            if (bodyMode == 'never-ending') return cancellationGate.future;
            if (bodyMode == 'cancel-fails') {
              return Future<void>.error(
                StateError('private cancellation failure'),
              );
            }
          },
        );
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          _Session(),
          client: MockClient.streaming((request, _) async {
            calls++;
            expect(request.method, 'GET');
            return http.StreamedResponse(body.stream, status);
          }),
        );
        try {
          await expectLater(
            optionalResume(api, item).timeout(const Duration(seconds: 1)),
            throwsA(
              isA<ApiFailure>()
                  .having((error) => error.status, 'known status', status)
                  .having(
                    (error) => error.message.contains('private'),
                    'redacted body',
                    isFalse,
                  ),
            ),
          );
          expect(calls, 1);
          expect(cancellations, 1);
        } finally {
          cancellationGate.complete();
          await body.close();
          api.close();
        }
      });
    }
  }
  for (final mode in [
    'saved',
    '404',
    '503',
    '429',
    'malformed',
    'timeout',
    'transport',
    '401',
    '403',
    '400',
  ]) {
    test(
      'optional resume $mode preserves authority and never writes',
      () async {
        var calls = 0;
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          _Session(),
          client: MockClient((request) async {
            calls++;
            expect(request.method, 'GET');
            expect(request.url.path, '/prosody/v1/items/item/progress');
            expect(request.url.queryParameters['renditionId'], 'rendition');
            if (mode == 'timeout') throw TimeoutException('test');
            if (mode == 'transport') throw http.ClientException('test');
            if (mode == 'malformed') return http.Response('{}', 200);
            final status = int.tryParse(mode);
            if (status != null) return http.Response('private detail', status);
            return http.Response(
              jsonEncode({
                'revision': 7,
                'positionSegment': 0,
                'positionMs': 400,
                'segments': [
                  {'durationMs': 1000},
                ],
              }),
              200,
            );
          }),
        );
        try {
          if (['401', '403', '400'].contains(mode)) {
            await expectLater(
              optionalResume(api, item),
              throwsA(
                isA<ApiFailure>().having(
                  (e) => e.status,
                  'status',
                  int.parse(mode),
                ),
              ),
            );
          } else {
            final result = await optionalResume(api, item);
            if (mode == 'saved') {
              expect(result?.positionMs, 400);
              expect(result?.revision, 7);
            } else {
              expect(result, isNull);
            }
          }
          expect(calls, 1);
        } finally {
          api.close();
        }
      },
    );
  }
}
