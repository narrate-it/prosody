import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';

class _Session implements AccessSession {
  String token = 'original-session';
  bool expired = false;
  @override
  Future<String> accessToken() async {
    if (expired) throw StateError('Synthetic expired session');
    return token;
  }

  @override
  Future<void> signOut() async => token = 'later-session';
}

Map<String, Object?> _page(String id, {String? next}) => {
  'feeds': [
    {'id': id, 'name': id, 'kind': 'team', 'canManage': false},
  ],
  'nextCursor': next,
};
http.Response _json(Object value) => http.Response(jsonEncode(value), 200);

void main() {
  for (final duringFinalPage in [false, true]) {
    test(
      'directory refuses session change during ${duringFinalPage ? 'final' : 'first'} page',
      () async {
        final session = _Session();
        final requests = <http.Request>[];
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            requests.add(request);
            if (requests.length == 1) {
              await session.signOut();
              return _json(
                _page(
                  'original-private-feed',
                  next: duringFinalPage ? null : 'next',
                ),
              );
            }
            return _json(_page('later-private-feed'));
          }),
        );
        addTearDown(api.close);
        await expectLater(api.feeds(), throwsA(isA<ApiFailure>()));
        expect(requests.length, 1);
        expect(
          requests.single.headers['Authorization'],
          'Bearer original-session',
        );
      },
    );
  }

  for (final finalPage in [false, true]) {
    test(
      'directory expiry after ${finalPage ? 'final' : 'first'} page refuses result',
      () async {
        final session = _Session();
        var requests = 0;
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            requests++;
            session.expired = true;
            return _json(_page('private', next: finalPage ? null : 'next'));
          }),
        );
        addTearDown(api.close);
        await expectLater(api.feeds(), throwsStateError);
        expect(requests, 1);
      },
    );
  }

  for (final empty in [false, true]) {
    test(
      'same-session directory remains immutable and bounded: empty=$empty',
      () async {
        final session = _Session();
        final requests = <http.Request>[];
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            requests.add(request);
            expect(request.followRedirects, isFalse);
            expect(request.url.path, '/prosody/v1/feeds');
            expect(request.headers['Authorization'], 'Bearer original-session');
            if (empty) return _json({'feeds': [], 'nextCursor': null});
            return _json(
              _page(
                requests.length == 1 ? 'first' : 'second',
                next: requests.length == 1 ? 'next' : null,
              ),
            );
          }),
        );
        addTearDown(api.close);
        final feeds = await api.feeds();
        expect(feeds.map((f) => f.id), empty ? isEmpty : ['first', 'second']);
        expect(() => feeds.clear(), throwsUnsupportedError);
        expect(requests.length, empty ? 1 : 2);
        if (!empty) {
          expect(requests.last.url.queryParameters, {'cursor': 'next'});
        }
      },
    );
  }

  for (final malformed in [false, true]) {
    test(
      'directory never returns partial results or retries: malformed=$malformed',
      () async {
        final session = _Session();
        var requests = 0;
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            requests++;
            if (requests == 1) return _json(_page('first', next: 'next'));
            return malformed
                ? _json({'feeds': 'invalid'})
                : http.Response('private server detail', 403);
          }),
        );
        addTearDown(api.close);
        await expectLater(
          api.feeds(),
          throwsA(
            malformed
                ? isA<FormatException>()
                : isA<ApiFailure>()
                      .having((e) => e.status, 'status', 403)
                      .having(
                        (e) => e.message,
                        'safe message',
                        isNot(contains('private server detail')),
                      ),
          ),
        );
        expect(requests, 2);
      },
    );
  }
}
