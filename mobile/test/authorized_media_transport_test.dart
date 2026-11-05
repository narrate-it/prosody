import 'dart:async';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:video_player/video_player.dart';
import 'package:prosody/data/models.dart';
import 'package:prosody/playback/handler.dart';
import 'package:prosody/data/authorized_media.dart';
import 'package:prosody/playback/authorized_media_buffer.dart';

final asset = Uri.parse('https://media.example/private/one');

class Headers implements HttpHeaders {
  final values = <String, String>{};
  @override
  void set(String name, Object value, {bool preserveHeaderCase = false}) {
    values[name.toLowerCase()] = value.toString();
  }

  @override
  String? value(String name) => values[name.toLowerCase()];
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class Response extends Stream<List<int>> implements HttpClientResponse {
  @override
  final int statusCode;
  @override
  final int contentLength;
  @override
  final Headers headers = Headers();
  final Stream<List<int>> body;
  Response({
    this.statusCode = 206,
    this.contentLength = 3,
    String mime = 'audio/mpeg',
    String etag = '"one"',
    String range = 'bytes 0-2/3',
    List<int> bytes = const [1, 2, 3],
    Stream<List<int>>? stream,
  }) : body = stream ?? Stream.value(bytes) {
    headers.set('content-type', mime);
    headers.set('etag', etag);
    headers.set('content-range', range);
  }
  @override
  StreamSubscription<List<int>> listen(
    void Function(List<int>)? onData, {
    Function? onError,
    void Function()? onDone,
    bool? cancelOnError,
  }) => body.listen(
    onData,
    onError: onError,
    onDone: onDone,
    cancelOnError: cancelOnError,
  );
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class Request implements HttpClientRequest {
  @override
  final Headers headers = Headers();
  @override
  bool followRedirects = true;
  final Response response;
  Request(this.response);
  @override
  Future<HttpClientResponse> close() async => response;
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class Client implements HttpClient {
  final Request request;
  bool closed = false;
  String? method;
  Uri? uri;
  Client(Response response) : request = Request(response);
  @override
  set autoUncompress(bool value) {}
  @override
  set connectionTimeout(Duration? value) {}
  @override
  Future<HttpClientRequest> openUrl(String method, Uri uri) async {
    this.method = method;
    this.uri = uri;
    return request;
  }

  @override
  void close({bool force = false}) {
    closed = true;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class Harness {
  final List<Client> clients = [];
  final List<Response> responses;
  int tokens = 0;
  late final AuthorizedMediaTransport transport;
  Harness(this.responses, {Future<MediaBearer> Function()? bearer}) {
    transport = AuthorizedMediaTransport(
      admittedAssets: {asset: AdmittedMediaFormat.mp3},
      bearer:
          bearer ??
          () async => MediaBearer('token${++tokens}', DateTime.utc(2099)),
      clientFactory: () {
        final client = Client(responses.removeAt(0));
        clients.add(client);
        return client;
      },
    );
  }
}

Response head({String mime = 'audio/mpeg'}) => Response(
  statusCode: 200,
  stream: const Stream<List<int>>.empty(),
  mime: mime,
);

class BrokenSink implements IOSink {
  final bool flushFails;
  final Object primary;
  BrokenSink(this.flushFails, this.primary);
  @override
  void add(List<int> data) {}
  @override
  Future<void> flush() async {
    if (flushFails) {
      throw primary;
    }
  }

  @override
  Future<void> close() async => throw primary;
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class Storage extends MediaBufferStorage {
  final List<Directory> created = [];
  bool broken = true;
  final bool flushFails;
  final Object primary = StateError('write failure');
  Storage({required this.flushFails});
  @override
  Future<Directory> createDirectory() async {
    final directory = await super.createDirectory();
    created.add(directory);
    return directory;
  }

  @override
  IOSink openWrite(File file) =>
      broken ? BrokenSink(flushFails, primary) : super.openWrite(file);
}

class GatedVideo extends Fake implements VideoPlayerController {
  final initializationStarted = Completer<void>();
  final initialization = Completer<void>();
  final disposalStarted = Completer<void>();
  final disposal = Completer<void>();
  int disposeCalls = 0;
  @override
  VideoPlayerValue get value => const VideoPlayerValue(
    duration: Duration(seconds: 1),
    isInitialized: true,
  );
  @override
  void addListener(void Function() listener) {}
  @override
  Future<void> initialize() {
    initializationStarted.complete();
    return initialization.future;
  }

  @override
  Future<void> dispose() {
    disposeCalls++;
    if (!disposalStarted.isCompleted) {
      disposalStarted.complete();
    }
    return disposal.future;
  }
}

class QuietAudio extends Fake implements audio.AudioPlayer {
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

Briefing item(String id, {bool video = false}) => Briefing(
  id: id,
  renditionId: '$id-r',
  title: id,
  agent: 'Agent',
  status: 'ready',
  source: '',
  script: '',
  segments: [
    Segment(
      id: '$id-s',
      kind: video ? 'video' : 'image',
      duration: video ? const Duration(seconds: 1) : Duration.zero,
      media: video ? asset : null,
      images: video ? const [] : [asset],
    ),
  ],
);

Future<T> bounded<T>(Future<T> operation, String stage) => operation.timeout(
  const Duration(seconds: 3),
  onTimeout: () => throw TimeoutException('Media test stalled: $stage'),
);

void main() {
  for (final initializing in [true, false]) {
    test(
      'concurrent transition and revoke await decoder disposal initializing=$initializing before file deletion',
      () async {
        final h = Harness([head(), Response()]);
        final controller = GatedVideo();
        if (!initializing) {
          controller.initialization.complete();
        }
        late ProsodyPlayback playback;
        final buffer = AuthorizedMediaBuffer(
          transport: h.transport,
          maxAssetBytes: 3,
          onAccessEnded: () => playback.stop(),
        );
        late File decoderFile;
        playback = ProsodyPlayback(
          audioPlayer: QuietAudio(),
          authorizedMedia: buffer,
          videoFactory: (uri, file, options) {
            decoderFile = file!;
            return controller;
          },
        );
        addTearDown(() async {
          if (!controller.disposal.isCompleted) {
            controller.disposal.complete();
          }
          if (!controller.initialization.isCompleted) {
            controller.initialization.complete();
          }
          await bounded(playback.dispose(), 'gated decoder teardown');
        });
        await bounded(
          playback.replaceFeed([
            item('first'),
            item('video', video: true),
            item('next'),
          ]),
          'initial visual feed',
        );
        await bounded(playback.pause(), 'pause visual');
        final selecting = playback.select(1, autoplay: false);
        await bounded(
          controller.initializationStarted.future,
          'decoder initialization started',
        );
        if (!initializing) {
          await bounded(selecting, 'selected decoder settles');
        }
        final transitioning = playback.select(2, autoplay: false);
        await bounded(
          controller.disposalStarted.future,
          'decoder disposal started',
        );
        var revoked = false;
        final revoking = buffer.close().then((_) {
          revoked = true;
        });
        await Future<void>.delayed(Duration.zero);
        expect(revoked, isFalse);
        expect(await decoderFile.exists(), isTrue);
        expect(controller.disposeCalls, 1);
        controller.disposal.complete();
        await bounded(revoking, 'revoke after disposal gate');
        expect(await decoderFile.exists(), isFalse);
        if (initializing) {
          controller.initialization.complete();
        }
        await bounded(selecting, 'selected decoder settles');
        await bounded(transitioning, 'stale transition settles');
        expect(controller.disposeCalls, 1);
      },
    );
  }

  test(
    'successful release completes and concurrent release shares cleanup',
    () async {
      final h = Harness([head(), Response()]);
      final buffer = AuthorizedMediaBuffer(
        transport: h.transport,
        maxAssetBytes: 3,
        onAccessEnded: () async {},
      );
      final file = await bounded(
        buffer.prepare(asset),
        'release control prepare',
      );
      await bounded(
        Future.wait([buffer.release(file), buffer.release(file)]),
        'shared release completion',
      );
      expect(await file.exists(), isFalse);
      await bounded(buffer.release(file), 'idempotent release');
      await bounded(buffer.close(), 'close after direct release');
    },
  );

  test('HEAD body remains rejected even when the chunk is empty', () async {
    final h = Harness([Response(statusCode: 200, bytes: [])]);
    await expectLater(h.transport.head(asset), throwsFormatException);
  });

  test(
    'HEAD and 206 control uses fresh bearer and disables redirects',
    () async {
      final h = Harness([head(), Response()]);
      final metadata = await h.transport.head(asset);
      final bytes = await h.transport.range(asset, 0, 2);
      expect(metadata.format, AdmittedMediaFormat.mp3);
      expect(bytes.bytes, [1, 2, 3]);
      expect(h.tokens, 2);
      expect(h.clients.map((c) => c.method), ['HEAD', 'GET']);
      expect(h.clients.map((c) => c.request.headers.value('authorization')), [
        'Bearer token1',
        'Bearer token2',
      ]);
      expect(
        h.clients.every((c) => !c.request.followRedirects && c.closed),
        isTrue,
      );
      expect(h.clients.last.request.headers.value('range'), 'bytes=0-2');
    },
  );

  for (final entry in <String, Response Function()>{
    'wrong MIME': () => Response(mime: 'text/html'),
    'redirect': () => Response(statusCode: 302),
    'wrong range': () => Response(range: 'bytes 1-3/4'),
    'wrong length': () => Response(contentLength: 4),
    'truncated': () => Response(bytes: [1, 2]),
    'oversized': () => Response(bytes: [1, 2, 3, 4]),
    'weak ETag': () => Response(etag: 'W/"one"'),
  }.entries) {
    test('rejects ${entry.key}', () async {
      final h = Harness([entry.value()]);
      await expectLater(h.transport.range(asset, 0, 2), throwsFormatException);
      expect(h.clients.single.closed, isTrue);
    });
  }

  test('HEAD rejects MIME before any local file creation', () async {
    final h = Harness([head(mime: 'text/html')]);
    final storage = Storage(flushFails: true);
    final buffer = AuthorizedMediaBuffer(
      transport: h.transport,
      maxAssetBytes: 3,
      storage: storage,
      onAccessEnded: () async {},
    );
    await expectLater(buffer.prepare(asset), throwsFormatException);
    expect(storage.created, isEmpty);
    await buffer.close();
  });

  for (final response in [
    Response(etag: '"changed"'),
    Response(mime: 'video/mp4'),
  ]) {
    test(
      'buffer rejects representation ${response.headers.value('content-type')} ${response.headers.value('etag')} before handoff',
      () async {
        final h = Harness([head(), response]);
        final storage = Storage(flushFails: false)..broken = false;
        final buffer = AuthorizedMediaBuffer(
          transport: h.transport,
          maxAssetBytes: 3,
          storage: storage,
          onAccessEnded: () async {},
        );
        await expectLater(buffer.prepare(asset), throwsA(anything));
        for (final directory in storage.created) {
          expect(await directory.exists(), isFalse);
        }
        await buffer.close();
      },
    );
  }

  for (final flushFails in [true, false]) {
    test(
      'storage flushFails=$flushFails preserves primary error and releases reservation',
      () async {
        final h = Harness([head(), Response(), head(), Response()]);
        final storage = Storage(flushFails: flushFails);
        final buffer = AuthorizedMediaBuffer(
          transport: h.transport,
          maxAssetBytes: 3,
          storage: storage,
          onAccessEnded: () async {},
        );
        await expectLater(
          bounded(buffer.prepare(asset), 'failed write cleanup'),
          throwsA(same(storage.primary)),
        );
        expect(await storage.created.single.exists(), isFalse);
        storage.broken = false;
        final file = await bounded(
          buffer.prepare(asset),
          'reservation reuse prepare',
        ); // Same cap proves reservation released.
        expect(await file.readAsBytes(), [1, 2, 3]);
        expect(file.path.endsWith('.mp3'), isTrue);
        await bounded(buffer.close(), 'buffer close after reservation reuse');
        expect(await file.exists(), isFalse);
      },
    );
  }

  for (final status in [401, 403, 500]) {
    test(
      '$status distinguishes ended media scope from transient failure',
      () async {
        final h = Harness([head(), Response(), Response(statusCode: status)]);
        var stopped = false;
        final buffer = AuthorizedMediaBuffer(
          transport: h.transport,
          maxAssetBytes: 3,
          onAccessEnded: () async {
            stopped = true;
          },
        );
        final file = await buffer.prepare(asset);
        await expectLater(h.transport.head(asset), throwsA(anything));
        if (status == 500) {
          expect(buffer.isClosed, isFalse);
          expect(await file.exists(), isTrue);
        } else {
          await buffer.close();
          expect(stopped, isTrue);
          expect(await file.exists(), isFalse);
        }
        await buffer.close();
      },
    );
  }

  test(
    'denial stops decoders before deleting previously prepared media',
    () async {
      final h = Harness([head(), Response(), Response(statusCode: 403)]);
      final stopped = Completer<void>();
      late File file;
      final buffer = AuthorizedMediaBuffer(
        transport: h.transport,
        maxAssetBytes: 3,
        onAccessEnded: () async {
          expect(await file.exists(), isTrue);
          await stopped.future;
        },
      );
      file = await buffer.prepare(asset);
      await expectLater(
        h.transport.head(asset),
        throwsA(isA<MediaAccessEnded>()),
      );
      expect(buffer.isClosed, isTrue);
      expect(await file.exists(), isTrue);
      stopped.complete();
      await buffer.close();
      expect(await file.exists(), isFalse);
    },
  );

  for (final revoke in [false, true]) {
    test(
      'revoke=$revoke while bearer is pending prevents network use',
      () async {
        final token = Completer<MediaBearer>();
        final h = Harness([], bearer: () => token.future);
        final pending = h.transport.head(asset);
        final rejected = expectLater(pending, throwsA(isA<MediaAccessEnded>()));
        if (revoke) {
          h.transport.revoke();
        } else {
          h.transport.cancelRequests();
        }
        token.complete(MediaBearer('secret', DateTime.utc(2099)));
        await rejected;
        expect(h.clients, isEmpty);
      },
    );
  }

  test('expired bearer fails before opening a client', () async {
    final h = Harness(
      [],
      bearer: () async => MediaBearer('secret', DateTime.utc(2000)),
    );
    await expectLater(
      h.transport.head(asset),
      throwsA(isA<MediaAccessEnded>()),
    );
    expect(h.clients, isEmpty);
  });

  for (final revoke in [false, true]) {
    test(
      'revoke=$revoke during stalled body rejects handoff and closes client',
      () async {
        final body = StreamController<List<int>>();
        final h = Harness([Response(stream: body.stream)]);
        final future = h.transport.range(asset, 0, 2);
        final rejected = expectLater(future, throwsA(isA<MediaAccessEnded>()));
        // A microtask checkpoint lets request.close resolve without a wall clock wait.
        await Future<void>.delayed(Duration.zero);
        if (revoke) {
          h.transport.revoke();
        } else {
          h.transport.cancelRequests();
        }
        await rejected; // No next body event is needed to cancel.
        await body.close();
        expect(h.clients.single.closed, isTrue);
      },
    );
  }
}
