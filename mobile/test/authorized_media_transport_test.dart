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
  Harness(
    this.responses, {
    Future<MediaBearer> Function()? bearer,
    Map<Uri, AdmittedMediaFormat>? admittedAssets,
    DateTime Function()? clock,
  }) {
    transport = AuthorizedMediaTransport(
      admittedAssets: admittedAssets ?? {asset: AdmittedMediaFormat.mp3},
      clock: clock,
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
  int failDisposals = 0;
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
    if (failDisposals > 0) {
      failDisposals--;
      return Future.error(StateError('decoder disposal failed'));
    }
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

class FileAudio extends QuietAudio {
  final paths = <String>[];
  int failStops = 0;
  @override
  Future<Duration?> setFilePath(
    String filePath, {
    Duration? initialPosition,
    bool preload = true,
    dynamic tag,
  }) async {
    paths.add(filePath);
    return const Duration(seconds: 1);
  }

  @override
  Future<void> stop() async {
    if (failStops > 0) {
      failStops--;
      throw StateError('injected audio stop failure');
    }
  }
}

class BlockedStopAudio extends QuietAudio {
  bool blockNext = false;
  final started = Completer<void>();
  final gate = Completer<void>();
  @override
  Future<void> stop() {
    if (blockNext) {
      blockNext = false;
      started.complete();
      return gate.future;
    }
    return Future.value();
  }
}

class ManualTimer implements Timer {
  final void Function() callback;
  @override
  bool isActive = true;
  @override
  int tick = 0;
  ManualTimer(this.callback);
  void fire() {
    if (isActive) {
      isActive = false;
      tick++;
      callback();
    }
  }

  @override
  void cancel() {
    isActive = false;
  }
}

class FailingDeleteStorage extends MediaBufferStorage {
  String? failOncePath;
  final attempted = <String>[];
  @override
  Future<void> deleteDirectory(Directory directory) async {
    attempted.add(directory.path);
    if (directory.path == failOncePath) {
      failOncePath = null;
      throw const FileSystemException('injected delete failure');
    }
    await super.deleteDirectory(directory);
  }
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

Briefing audioItem(String id) => Briefing(
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
      kind: 'audio',
      duration: const Duration(seconds: 1),
      media: asset,
    ),
  ],
);

Future<T> bounded<T>(Future<T> operation, String stage) => operation.timeout(
  const Duration(seconds: 3),
  onTimeout: () => throw TimeoutException('Media test stalled: $stage'),
);

void main() {
  for (final failStop in [false, true]) {
    test(
      'audio release failure retains one-cap retry ownership after stop failure=$failStop',
      () async {
        final h = Harness([head(), Response(), head(), Response()]);
        final storage = FailingDeleteStorage();
        final audioPlayer = FileAudio();
        late ProsodyPlayback playback;
        final buffer = AuthorizedMediaBuffer(
          transport: h.transport,
          maxAssetBytes: 3,
          storage: storage,
          onAccessEnded: () => playback.stop(),
        );
        playback = ProsodyPlayback(
          audioPlayer: audioPlayer,
          authorizedMedia: buffer,
        );
        addTearDown(() async {
          await bounded(playback.dispose(), 'audio retirement teardown');
        });
        await bounded(
          playback.replaceFeed([
            item('first'),
            audioItem('old'),
            audioItem('next'),
          ]),
          'audio retirement feed',
        );
        await bounded(playback.pause(), 'audio retirement pause');
        await bounded(playback.select(1, autoplay: false), 'audio old ready');
        final old = File(audioPlayer.paths.single);
        storage.failOncePath = old.parent.path;
        if (failStop) {
          audioPlayer.failStops = 1;
          await bounded(
            playback.select(2, autoplay: false),
            'injected audio stop failure',
          );
          expect(playback.failure.value, isNotNull);
          expect(await old.exists(), isTrue);
        }
        await bounded(
          playback.select(2, autoplay: false),
          'audio transient delete failure',
        );
        expect(playback.failure.value, isNotNull);
        expect(audioPlayer.paths, hasLength(1));
        expect(await old.exists(), isTrue);
        // Neither a close nor a directory sweep may recover this reservation.
        await bounded(
          playback.select(2, autoplay: false),
          'audio ordinary selection retry',
        );
        expect(buffer.isClosed, isFalse);
        expect(audioPlayer.paths, hasLength(2));
        expect(await old.exists(), isFalse);
        expect(await File(audioPlayer.paths.last).exists(), isTrue);
        expect(playback.current?.id, 'next');
        expect(playback.failure.value, isNull);
      },
    );
  }

  test(
    'conservative whole-scope expiry cannot be extended by a later bearer',
    () async {
      final current = DateTime.utc(2090);
      var tokens = 0;
      final h = Harness(
        [head(), Response(), head(), Response()],
        clock: () => current,
        bearer: () async =>
            MediaBearer('t${tokens++}', current.add(Duration(minutes: tokens))),
      );
      late ManualTimer expiry;
      var schedules = 0;
      final buffer = AuthorizedMediaBuffer(
        transport: h.transport,
        maxAssetBytes: 3,
        onAccessEnded: () async {},
        scheduleExpiry: (duration, callback) {
          schedules++;
          expiry = ManualTimer(callback);
          return expiry;
        },
      );
      final first = await bounded(
        buffer.prepare(asset),
        'first expiry preparation',
      );
      await bounded(buffer.release(first), 'release first expiry file');
      final second = await bounded(
        buffer.prepare(asset),
        'later bearer preparation',
      );
      expect(schedules, 1); // Later credentials never renew this whole scope.
      expiry.fire();
      await bounded(buffer.close(), 'earliest scope expiry');
      expect(await second.exists(), isFalse);
      await expectLater(
        buffer.prepare(asset),
        throwsA(isA<MediaAccessEnded>()),
      );
      // A separately admitted new grant gets an entirely new transport/buffer.
      final replacement = Harness([head(), Response()]);
      final newBuffer = AuthorizedMediaBuffer(
        transport: replacement.transport,
        maxAssetBytes: 3,
        onAccessEnded: () async {},
      );
      final newFile = await bounded(
        newBuffer.prepare(asset),
        'new admitted scope',
      );
      expect(await newFile.exists(), isTrue);
      await bounded(newBuffer.close(), 'new scope cleanup');
    },
  );

  test(
    'close attempts unrelated directories and permits explicit deletion retry',
    () async {
      final h = Harness([head(), Response(), head(), Response()]);
      final storage = FailingDeleteStorage();
      final buffer = AuthorizedMediaBuffer(
        transport: h.transport,
        maxAssetBytes: 6,
        storage: storage,
        onAccessEnded: () async {},
      );
      final first = await bounded(
        buffer.prepare(asset),
        'first delete retry file',
      );
      final second = await bounded(
        buffer.prepare(asset),
        'second delete retry file',
      );
      storage.failOncePath = first.parent.path;
      await expectLater(
        bounded(buffer.close(), 'first deletion attempt'),
        throwsA(isA<MediaBufferCleanupFailed>()),
      );
      expect(await first.exists(), isTrue);
      expect(await second.exists(), isFalse);
      expect(
        storage.attempted,
        containsAll([first.parent.path, second.parent.path]),
      );
      await bounded(buffer.close(), 'explicit deletion retry');
      expect(await first.exists(), isFalse);
    },
  );

  test('failed shutdown preserves protected file while cleaning unrelated file, then retries', () async {
    final h = Harness([head(), Response(), head(), Response()]);
    var attempts = 0;
    late File protected;
    late AuthorizedMediaBuffer buffer;
    buffer = AuthorizedMediaBuffer(
      transport: h.transport,
      maxAssetBytes: 6,
      onAccessEnded: () async {
        if (++attempts == 1) {
          throw StateError('decoder stop failed');
        }
        buffer.unprotect(protected);
      },
    );
    protected = await bounded(buffer.prepare(asset), 'protected preparation');
    buffer.protect(protected);
    final unrelated = await bounded(
      buffer.prepare(asset),
      'unrelated preparation',
    );
    await expectLater(
      bounded(buffer.close(), 'failed shutdown attempt'),
      throwsA(isA<MediaBufferCleanupFailed>()),
    );
    expect(await protected.exists(), isTrue);
    expect(await unrelated.exists(), isFalse);
    await bounded(buffer.close(), 'explicit shutdown retry');
    expect(await protected.exists(), isFalse);
    expect(attempts, 2);
  });

  test(
    'revoke first cannot be superseded by selection while audio shutdown waits',
    () async {
      final h = Harness([head(), Response()]);
      final controller = GatedVideo();
      controller.initialization.complete();
      controller.disposal.complete();
      final audioPlayer = BlockedStopAudio();
      late ProsodyPlayback playback;
      final buffer = AuthorizedMediaBuffer(
        transport: h.transport,
        maxAssetBytes: 3,
        onAccessEnded: () => playback.stop(),
      );
      late File decoderFile;
      playback = ProsodyPlayback(
        audioPlayer: audioPlayer,
        authorizedMedia: buffer,
        videoFactory: (uri, file, options) {
          decoderFile = file!;
          return controller;
        },
      );
      addTearDown(() async {
        if (!audioPlayer.gate.isCompleted) {
          audioPlayer.gate.complete();
        }
        await bounded(playback.dispose(), 'revoke-first teardown');
      });
      await bounded(
        playback.replaceFeed([
          item('first'),
          item('video', video: true),
          item('next'),
        ]),
        'revoke-first feed',
      );
      await bounded(playback.pause(), 'revoke-first pause');
      await bounded(
        playback.select(1, autoplay: false),
        'revoke-first video ready',
      );
      audioPlayer.blockNext = true;
      var ended = false;
      final revoking = buffer.close().then((_) {
        ended = true;
      });
      await bounded(
        audioPlayer.started.future,
        'revoke audio shutdown started',
      );
      await bounded(
        playback.select(2, autoplay: false),
        'selection during terminal shutdown',
      );
      expect(ended, isFalse);
      expect(await decoderFile.exists(), isTrue);
      audioPlayer.gate.complete();
      await bounded(revoking, 'revoke-first shutdown completion');
      expect(await decoderFile.exists(), isFalse);
      expect(playback.current, isNull);
      await bounded(playback.dispose(), 'revoke-first dispose');
    },
  );

  test('ordinary selection retry releases failed decoder file before reusing one-asset cap', () async {
    final h = Harness([head(), Response(), head(), Response()]);
    final firstController = GatedVideo();
    firstController.initialization.complete();
    firstController.disposal.complete();
    final nextController = GatedVideo();
    nextController.initialization.complete();
    nextController.disposal.complete();
    final files = <File>[];
    late ProsodyPlayback playback;
    final buffer = AuthorizedMediaBuffer(
      transport: h.transport,
      maxAssetBytes: 3,
      onAccessEnded: () => playback.stop(),
    );
    playback = ProsodyPlayback(
      audioPlayer: QuietAudio(),
      authorizedMedia: buffer,
      videoFactory: (uri, file, options) {
        files.add(file!);
        return files.length == 1 ? firstController : nextController;
      },
    );
    addTearDown(() async {
      await bounded(playback.dispose(), 'one-cap retry teardown');
    });
    await bounded(
      playback.replaceFeed([
        item('first'),
        item('old', video: true),
        item('next', video: true),
      ]),
      'one-cap feed',
    );
    await bounded(playback.pause(), 'one-cap pause');
    await bounded(playback.select(1, autoplay: false), 'one-cap old decoder');
    firstController.failDisposals = 1;
    await bounded(
      playback.select(2, autoplay: false),
      'one-cap failed transition',
    );
    expect(playback.failure.value, isNotNull);
    expect(files, hasLength(1));
    expect(await files.first.exists(), isTrue);
    // No close/sweep occurs: this ordinary selection must recover capacity.
    await bounded(
      playback.select(2, autoplay: false),
      'one-cap ordinary retry',
    );
    expect(buffer.isClosed, isFalse);
    expect(files, hasLength(2));
    expect(await files.first.exists(), isFalse);
    expect(await files.last.exists(), isTrue);
    expect(playback.current?.id, 'next');
    expect(playback.failure.value, isNull);
    expect(firstController.disposeCalls, 2);
    expect(nextController.disposeCalls, 0);
  });

  test('decoder disposal failure surfaces failure and allows protected cleanup retry', () async {
    final h = Harness([head(), Response(), head(), Response()]);
    final controller = GatedVideo();
    controller.initialization.complete();
    controller.disposal.complete();
    late ProsodyPlayback playback;
    final buffer = AuthorizedMediaBuffer(
      transport: h.transport,
      maxAssetBytes: 6,
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
      await bounded(playback.dispose(), 'failed decoder teardown');
    });
    await bounded(
      playback.replaceFeed([
        item('first'),
        item('video', video: true),
        item('next'),
      ]),
      'failed decoder feed',
    );
    await bounded(playback.pause(), 'failed decoder pause');
    await bounded(playback.select(1, autoplay: false), 'failed decoder ready');
    final unrelated = await bounded(
      buffer.prepare(asset),
      'unrelated decoder file',
    );
    controller.failDisposals = 2;
    await bounded(
      playback.select(2, autoplay: false),
      'failed disposal transition',
    );
    expect(playback.failure.value, isNotNull);
    expect(await decoderFile.exists(), isTrue);
    await expectLater(
      bounded(buffer.close(), 'failed decoder shutdown'),
      throwsA(isA<MediaBufferCleanupFailed>()),
    );
    expect(await decoderFile.exists(), isTrue);
    expect(await unrelated.exists(), isFalse);
    await bounded(buffer.close(), 'decoder shutdown retry');
    expect(await decoderFile.exists(), isFalse);
    expect(controller.disposeCalls, 3);
    await bounded(playback.dispose(), 'failed decoder final dispose');
  });

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
