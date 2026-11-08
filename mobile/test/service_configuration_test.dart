import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/data/native_session.dart';
import 'package:prosody/data/service_configuration.dart';

void main() {
  const origin = 'https://api.example.invalid';
  const issuer = 'https://identity.example.invalid';
  const audience = 'https://api.example.invalid/prosody';

  test('absent release inputs keep native session disabled', () {
    expect(
      ServiceConfiguration.parse(origin: '', issuer: '', audience: ''),
      isNull,
    );
  });

  test(
    'explicit configuration constructs real native client with exact binding',
    () {
      final config = ServiceConfiguration.parse(
        origin: origin,
        issuer: issuer,
        audience: audience,
      )!;
      expect(config.origin.toString(), origin);
      final session = config.createSession();
      expect(session, isA<NativeSession>());
      expect(session.configuration.issuer.toString(), issuer);
      expect(session.configuration.audience, audience);
      expect(NativeSessionConfiguration.clientId, 'prosody-ios');
      expect(
        NativeSessionConfiguration.redirect,
        'run.prosody.prosody:/oauth/callback',
      );
    },
  );

  for (final missing in ['origin', 'issuer', 'audience']) {
    test('partial configuration refuses missing $missing', () {
      expect(
        () => ServiceConfiguration.parse(
          origin: missing == 'origin' ? '' : origin,
          issuer: missing == 'issuer' ? '' : issuer,
          audience: missing == 'audience' ? '' : audience,
        ),
        throwsFormatException,
      );
    });
  }

  for (final bad in [
    'http://api.example.invalid',
    'https://api.example.invalid/',
    'https://user@api.example.invalid',
    'https://api.example.invalid?x',
    'https://api.example.invalid#x',
    ' https://api.example.invalid',
    'https://api.example.invalid:443',
  ]) {
    test('invalid API origin refuses $bad', () {
      expect(
        () => ServiceConfiguration.parse(
          origin: bad,
          issuer: issuer,
          audience: audience,
        ),
        throwsFormatException,
      );
    });
    test('invalid issuer origin refuses $bad', () {
      expect(
        () => ServiceConfiguration.parse(
          origin: origin,
          issuer: bad,
          audience: audience,
        ),
        throwsFormatException,
      );
    });
  }

  for (final bad in [
    'https://api.example.invalid',
    'http://api.example.invalid/resource',
    'https://user@api.example.invalid/resource',
    'https://api.example.invalid/resource?x',
    'https://api.example.invalid/resource#x',
    'https://api.example.invalid/a/../b',
    'https://api.example.invalid/a//b',
    'https://api.example.invalid/%2f',
    'https://api.example.invalid/private value',
  ]) {
    test('invalid audience refuses $bad with safe diagnostics', () {
      expect(
        () => ServiceConfiguration.parse(
          origin: origin,
          issuer: issuer,
          audience: bad,
        ),
        throwsA(
          isA<FormatException>().having(
            (e) => e.message,
            'message',
            'Invalid service configuration',
          ),
        ),
      );
    });
  }

  test('explicit root resource remains valid', () {
    expect(
      ServiceConfiguration.parse(
        origin: origin,
        issuer: issuer,
        audience: '$origin/',
      )!.session.audience,
      '$origin/',
    );
  });
}
