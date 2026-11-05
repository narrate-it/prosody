import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/data/authorized_media.dart';
import 'package:prosody/playback/authorized_media_buffer.dart';

void main() {
  final asset = Uri.parse('https://media.example/private/asset-1');
  MediaBearer token() => MediaBearer('opaque', DateTime.utc(2099));

  test('admission rejects URL credentials, queries and path ambiguity', () {
    for (final value in [
      'http://media.example/private/asset-1',
      'https://user@media.example/private/asset-1',
      'https://media.example/private/asset-1?token=secret',
      'https://media.example/private/asset-1#part',
      'https://media.example/private/%2Fother',
    ]) {
      expect(
        () => AuthorizedMediaTransport(
          admittedAssets: {Uri.parse(value): AdmittedMediaFormat.mp3},
          bearer: () async => token(),
        ),
        throwsFormatException,
      );
    }
  });

  test('same origin does not authorize another endpoint or query', () async {
    var resolved = 0;
    final transport = AuthorizedMediaTransport(
      admittedAssets: {asset: AdmittedMediaFormat.mp3},
      bearer: () async {
        resolved++;
        return token();
      },
    );
    await expectLater(
      transport.head(Uri.parse('https://media.example/admin')),
      throwsA(isA<MediaAccessEnded>()),
    );
    await expectLater(
      transport.head(Uri.parse('$asset?token=secret')),
      throwsFormatException,
    );
    expect(resolved, 0);
  });

  test('only positive closed ranges up to one MiB are admitted', () {
    final transport = AuthorizedMediaTransport(
      admittedAssets: {asset: AdmittedMediaFormat.mp3},
      bearer: () async => token(),
    );
    for (final range in [
      [-1, 0],
      [2, 1],
      [0, 1024 * 1024],
    ]) {
      expect(
        () => transport.range(asset, range[0], range[1]),
        throwsArgumentError,
      );
    }
  });

  test('revocation rejects new requests before resolving a bearer', () async {
    var resolved = 0;
    final transport = AuthorizedMediaTransport(
      admittedAssets: {asset: AdmittedMediaFormat.mp3},
      bearer: () async {
        resolved++;
        return token();
      },
    );
    transport.revoke();
    await expectLater(transport.head(asset), throwsA(isA<MediaAccessEnded>()));
    expect(resolved, 0);
  });

  test('buffer refuses an unbounded whole-file policy', () {
    final transport = AuthorizedMediaTransport(
      admittedAssets: {asset: AdmittedMediaFormat.mp3},
      bearer: () async => token(),
    );
    expect(
      () => AuthorizedMediaBuffer(
        transport: transport,
        maxAssetBytes: 1024 * 1024 * 1024,
        onAccessEnded: () async {},
      ),
      throwsArgumentError,
    );
  });
}
