import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/preview/live_provider.dart';

Map<String, Object> catalog(bool speech) => {'data': [
  {'id': speech ? LiveProvider.speechModel : LiveProvider.textModel,
    'architecture': {'input_modalities': ['text'],
      'output_modalities': [speech ? 'speech' : 'text']},
    'supported_voices': ['laura'],
    'pricing': {'prompt': speech ? '0.00004' : '0.0000004',
      'completion': speech ? '0' : '0.0000016'}}
]};
http.Response chat([String script = 'A concise briefing.', String finish = 'stop']) =>
  http.Response(jsonEncode({'id': 'gen-rewrite', 'choices': [
    {'finish_reason': finish, 'message': {'content': script}}
  ]}), 200);

void main() {
  test('valid catalogs precede reservation and single rewrite then speech', () async {
    final events = <String>[];
    final provider = LiveProvider(client: MockClient((request) async {
      expect(request.followRedirects, isFalse);
      expect(request.headers['Authorization'], 'Bearer test-key');
      if (request.url.path.endsWith('/models')) {
        final modality = request.url.queryParameters['output_modalities'];
        events.add(modality!);
        return http.Response(jsonEncode(catalog(modality == 'speech')), 200);
      }
      expect(events, contains('reserve'));
      final body = jsonDecode(request.body) as Map;
      if (request.url.path.endsWith('/chat/completions')) {
        events.add('rewrite');
        expect(body['max_tokens'], 400);
        expect(body['provider'], {'allow_fallbacks': false,
          'require_parameters': true,
          'max_price': {'prompt': 0.4, 'completion': 1.6}});
        return chat();
      }
      events.add('speech-call');
      expect(body, {'model': LiveProvider.speechModel,
        'input': 'A concise briefing.', 'voice': 'laura', 'response_format': 'mp3'});
      return http.Response.bytes([73, 68, 51], 200,
        headers: {'content-type': 'audio/mpeg', 'x-generation-id': 'gen-speech'});
    }));
    addTearDown(provider.close);
    final result = await provider.generate(key: 'test-key', source: 'Source facts.',
      reserveAttempt: () async => events.add('reserve'));
    expect(events, ['text', 'speech', 'reserve', 'rewrite', 'speech-call']);
    expect(result.source, 'Source facts.');
    expect(result.script, 'A concise briefing.');
    expect(result.mp3, [73, 68, 51]);
    expect(result.rewriteGenerationId, 'gen-rewrite');
    expect(result.speechGenerationId, 'gen-speech');
  });

  for (final defect in ['price', 'voice', 'modality', 'missing-price']) {
    test('bad $defect catalog never reserves or generates', () async {
      var reserved = false;
      final provider = LiveProvider(client: MockClient((request) async {
        expect(request.method, 'GET');
        final data = catalog(request.url.queryParameters['output_modalities'] == 'speech');
        final model = (data['data'] as List).first as Map;
        if (request.url.queryParameters['output_modalities'] == 'speech') {
          switch (defect) {
            case 'price': model['pricing'] = {'prompt': '0.1', 'completion': '0'};
            case 'voice': model['supported_voices'] = ['other'];
            case 'modality': model['architecture'] = {'input_modalities': ['text'], 'output_modalities': ['audio']};
            case 'missing-price': model.remove('pricing');
          }
        }
        return http.Response(jsonEncode(data), 200);
      }));
      addTearDown(provider.close);
      await expectLater(provider.generate(key: 'test-key', source: 'Facts',
        reserveAttempt: () async { reserved = true; }), throwsA(isA<LiveProviderFailure>()));
      expect(reserved, isFalse);
    });
  }

  for (final source in ['x' * 4001, '😀' * 3001, String.fromCharCode(0xd800), ' ']) {
    test('invalid source fails before any request', () async {
      final provider = LiveProvider(client: MockClient((_) async => throw StateError('unexpected')));
      addTearDown(provider.close);
      await expectLater(provider.generate(key: 'test-key', source: source,
        reserveAttempt: () async => fail('unexpected reservation')), throwsA(isA<LiveProviderFailure>()));
    });
  }

  for (final defect in ['http', 'timeout', 'oversize-script', 'bytes', 'length', 'speech']) {
    test('$defect after reservation never retries and retains known receipt', () async {
      var posts = 0;
      var reservations = 0;
      final provider = LiveProvider(timeout: const Duration(milliseconds: 50),
        client: MockClient((request) async {
          if (request.method == 'GET') {
            return http.Response(jsonEncode(catalog(request.url.queryParameters['output_modalities'] == 'speech')), 200);
          }
          posts++;
          if (defect == 'http') return http.Response('SECRET provider error', 429);
          if (defect == 'timeout') return Completer<http.Response>().future;
          if (defect == 'oversize-script') return chat('x' * 2001);
          if (defect == 'bytes') return chat('😀' * 1001);
          if (defect == 'length') return chat('truncated', 'length');
          if (posts == 1) return chat();
          return http.Response('SECRET provider error', 500,
            headers: {'x-generation-id': 'gen-speech'});
        }));
      addTearDown(provider.close);
      try {
        await provider.generate(key: 'test-key', source: 'Facts',
          reserveAttempt: () async { reservations++; });
        fail('expected failure');
      } on LiveProviderFailure catch (failure) {
        expect(failure.outcomeUnknown, isTrue);
        expect(failure.toString(), isNot(contains('SECRET')));
        if (defect == 'speech') {
          expect(failure.rewriteGenerationId, 'gen-rewrite');
          expect(failure.speechGenerationId, 'gen-speech');
        }
      }
      expect(posts, defect == 'speech' ? 2 : 1);
      expect(reservations, 1);
    });
  }

  test('speech body timeout retains both receipts without retry', () async {
    var posts = 0;
    final body = StreamController<List<int>>();
    final provider = LiveProvider(timeout: const Duration(milliseconds: 50),
      client: _StreamingClient((request) async {
        if (request.method == 'GET') {
          return http.StreamedResponse(Stream.value(utf8.encode(jsonEncode(
            catalog(request.url.queryParameters['output_modalities'] == 'speech')))), 200);
        }
        posts++;
        if (posts == 1) {
          return http.StreamedResponse(Stream.value(chat().bodyBytes), 200);
        }
        return http.StreamedResponse(body.stream, 200, headers: {
          'content-type': 'audio/mpeg', 'x-generation-id': 'gen-speech'});
      }));
    addTearDown(provider.close);
    await expectLater(provider.generate(key: 'test-key', source: 'Facts',
      reserveAttempt: () async {}), throwsA(isA<LiveProviderFailure>()
        .having((e) => e.rewriteGenerationId, 'rewrite receipt', 'gen-rewrite')
        .having((e) => e.speechGenerationId, 'speech receipt', 'gen-speech')));
    expect(posts, 2);
    await body.close();
  });

  test('failed durable reservation prevents all paid calls', () async {
    final provider = LiveProvider(client: MockClient((request) async {
      expect(request.method, 'GET');
      return http.Response(jsonEncode(catalog(request.url.queryParameters['output_modalities'] == 'speech')), 200);
    }));
    addTearDown(provider.close);
    await expectLater(provider.generate(key: 'test-key', source: 'Facts',
      reserveAttempt: () async => throw StateError('SECRET')),
      throwsA(isA<LiveProviderFailure>().having((e) => e.outcomeUnknown, 'dispatch', false)));
  });
}

class _StreamingClient extends http.BaseClient {
  _StreamingClient(this.respond);
  final Future<http.StreamedResponse> Function(http.BaseRequest) respond;
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) => respond(request);
}
