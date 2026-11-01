import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:prosody/data/api.dart';

class TestSession implements AccessSession {
  @override
  Future<String> accessToken() async => 'test-session';
  @override
  Future<void> signOut() async {}
}

void main() {
  test(
    'feed request binds bearer to HTTPS origin and refuses redirects',
    () async {
      final api = ProsodyApi(
        Uri.parse('https://prosody.example'),
        TestSession(),
        client: MockClient((r) async {
          expect(r.url.toString(), 'https://prosody.example/prosody/v1/feeds');
          expect(r.headers['Authorization'], 'Bearer test-session');
          expect(r.followRedirects, isFalse);
          return http.Response(
            '{"feeds":[{"id":"f","name":"Inbox","kind":"personal","canManage":true}]}',
            200,
          );
        }),
      );
      expect((await api.feeds()).single.name, 'Inbox');
      api.close();
    },
  );
  test('provider bodies are not exposed as error messages', () async {
    final api = ProsodyApi(
      Uri.parse('https://prosody.example'),
      TestSession(),
      client: MockClient((_) async => http.Response('secret stack trace', 500)),
    );
    await expectLater(
      api.feeds(),
      throwsA(
        isA<ApiFailure>().having(
          (e) => e.message,
          'message',
          isNot(contains('secret')),
        ),
      ),
    );
    api.close();
  });
  test('refuses an insecure API origin', () {
    expect(
      () => ProsodyApi(Uri.parse('http://example.test'), TestSession()),
      throwsArgumentError,
    );
  });
}
