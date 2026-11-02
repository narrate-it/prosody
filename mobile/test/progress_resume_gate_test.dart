import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/data/resume.dart';
import 'package:prosody/playback/handler.dart';
import 'package:prosody/playback/progress_recorder.dart';

class _Session implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-only';
  @override
  Future<void> signOut() async {}
}

class _Audio extends Fake implements audio.AudioPlayer {
  @override
  Future<void> stop() async {}
  @override
  Future<void> pause() async {}
  @override
  Future<void> dispose() async {}
  @override
  double get speed => 1;
  @override
  Duration get position => Duration.zero;
  @override
  Duration get bufferedPosition => Duration.zero;
  @override
  audio.ProcessingState get processingState => audio.ProcessingState.idle;
}

void main() {
  for (final explicitlyUsePosition in [false, true]) {
    test(
      'unknown resume protects saved position; explicit override=$explicitlyUsePosition',
      () async {
        final item = Briefing.fromJson({
          'id': 'item',
          'renditionId': 'rendition',
          'title': 'Title',
          'agent': 'Agent',
          'status': 'ready',
          'segments': [
            {
              'id': 'image',
              'kind': 'image',
              'durationMs': 0,
              'images': ['https://media.example/image'],
            },
            {
              'id': 'audio',
              'kind': 'audio',
              'durationMs': 1000,
              'url': 'https://media.example/audio',
            },
          ],
        });
        var reads = 0, writes = 0;
        var serverSegment = 1, serverPosition = 400, serverRevision = 9;
        Map<String, dynamic> saved() => {
          'revision': serverRevision,
          'positionSegment': serverSegment,
          'positionMs': serverPosition,
          'segments': [
            {'durationMs': 0},
            {'durationMs': 1000},
          ],
        };
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          _Session(),
          client: MockClient((request) async {
            if (request.method == 'GET') {
              if (++reads == 1) return http.Response('unavailable', 503);
              return http.Response(jsonEncode(saved()), 200);
            }
            expect(request.method, 'PUT');
            writes++;
            expect(explicitlyUsePosition, isTrue);
            final update = jsonDecode(request.body) as Map<String, dynamic>;
            expect(update['expectedRevision'], 9);
            expect(update['played'], isEmpty);
            expect(update['viewed'], isTrue);
            serverSegment = update['segment'] as int;
            serverPosition = update['positionMs'] as int;
            serverRevision++;
            return http.Response(jsonEncode(saved()), 200);
          }),
        );
        final handler = ProsodyPlayback(audioPlayer: _Audio());
        final recorder = ProgressRecorder(api, handler);
        try {
          final resume = await optionalResume(api, item);
          expect(resume.unknown, isTrue);
          recorder.setResumeWriteGate(resumeKnown: !resume.unknown);
          await handler.replaceFeed([item]);
          await handler.pause();
          handler.visibleImages.add('item/image/https://media.example/image');
          await Future<void>.delayed(const Duration(milliseconds: 600));
          // Completion/discontinuity callbacks must not bypass the unknown gate.
          handler.completed.value++;
          handler.discontinuity.value++;
          expect(writes, 0);
          expect(recorder.failure.value, contains('saved place'));
          if (explicitlyUsePosition) {
            await recorder.resumeSync();
            await Future<void>.delayed(const Duration(milliseconds: 600));
          }
          await recorder.close();
          expect(reads, explicitlyUsePosition ? 2 : 1);
          expect(writes, explicitlyUsePosition ? 1 : 0);
          expect(serverSegment, explicitlyUsePosition ? 0 : 1);
          expect(serverPosition, explicitlyUsePosition ? 0 : 400);
        } finally {
          await recorder.close();
          await handler.dispose();
          api.close();
        }
      },
      timeout: const Timeout(Duration(seconds: 10)),
    );
  }
}
