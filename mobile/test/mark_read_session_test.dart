import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';
import 'package:prosody/data/models.dart';

class _Session implements AccessSession {
  String token = 'original-session';
  int calls = 0;
  int? failAt, switchAfter;
  @override
  Future<String> accessToken() async {
    calls++;
    if (calls == failAt) throw StateError('Synthetic expired session');
    final captured = token;
    if (calls == switchAfter) token = 'later-session';
    return captured;
  }

  @override
  Future<void> signOut() async => token = 'later-session';
}

final _item = Briefing.fromJson({
  'id': 'shared-item',
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
Map<String, Object> _saved(int revision, {bool markedRead = false}) => {
  'revision': revision,
  'positionSegment': 0,
  'positionMs': 400,
  'segments': [
    {'durationMs': 1000},
  ],
  'markedRead': markedRead,
};
http.Response _json(Object body) => http.Response(jsonEncode(body), 200);

void main() {
  test(
    'mark read refuses a later session after a held revision read',
    () async {
      final session = _Session(), reached = Completer<void>();
      final response = Completer<http.Response>();
      final requests = <http.Request>[];
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        session,
        client: MockClient((request) async {
          requests.add(request);
          if (request.method == 'GET') {
            reached.complete();
            return response.future;
          }
          return _json(_saved(9, markedRead: true));
        }),
      );
      addTearDown(api.close);
      final result = expectLater(
        api.markRead(_item),
        throwsA(isA<ApiFailure>()),
      );
      await reached.future;
      await session.signOut();
      response.complete(_json(_saved(8)));
      await result;
      expect(requests.map((r) => r.method), ['GET']);
      expect(
        requests.single.headers['Authorization'],
        'Bearer original-session',
      );
    },
  );

  for (final switchAfterCheck in [false, true]) {
    test(
      'mark read retains original token and position: race=$switchAfterCheck',
      () async {
        final session = _Session()..switchAfter = switchAfterCheck ? 2 : null;
        final requests = <http.Request>[];
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            requests.add(request);
            return _json(
              _saved(
                request.method == 'GET' ? 8 : 9,
                markedRead: request.method == 'PUT',
              ),
            );
          }),
        );
        addTearDown(api.close);
        await api.markRead(_item);
        expect(requests.map((r) => r.method), ['GET', 'PUT']);
        expect(
          requests.map((r) => r.headers['Authorization']),
          everyElement('Bearer original-session'),
        );
        final update = jsonDecode(requests.last.body) as Map<String, dynamic>;
        expect(update['expectedRevision'], 8);
        expect(update['positionMs'], 400);
        expect(update['markedRead'], isTrue);
        expect(update['played'], isEmpty);
        expect(update['viewed'], isFalse);
        if (switchAfterCheck) expect(session.token, 'later-session');
      },
    );
  }

  for (final failAt in [1, 2]) {
    test(
      'expired mark-read session refuses subsequent transport: $failAt',
      () async {
        final session = _Session()..failAt = failAt;
        final methods = <String>[];
        final api = ProsodyApi(
          Uri.parse('https://prosody.example'),
          session,
          client: MockClient((request) async {
            methods.add(request.method);
            return _json(_saved(8));
          }),
        );
        addTearDown(api.close);
        await expectLater(api.markRead(_item), throwsStateError);
        expect(methods, failAt == 1 ? isEmpty : ['GET']);
      },
    );
  }

  for (final outcome in ['lost-write', 'invalid-read', 'unconfirmed-write']) {
    test('mark read does not retry $outcome', () async {
      final methods = <String>[];
      final cause = StateError('Synthetic lost mutation response');
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        _Session(),
        client: MockClient((request) async {
          methods.add(request.method);
          if (request.method == 'GET') {
            return _json(
              outcome == 'invalid-read' ? {'revision': 8} : _saved(8),
            );
          }
          if (outcome == 'lost-write') throw cause;
          return _json(_saved(9));
        }),
      );
      addTearDown(api.close);
      await expectLater(
        api.markRead(_item),
        throwsA(outcome == 'lost-write' ? same(cause) : isA<FormatException>()),
      );
      expect(methods, outcome == 'invalid-read' ? ['GET'] : ['GET', 'PUT']);
    });
  }
}
