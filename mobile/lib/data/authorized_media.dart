import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

/// Authority is supplied by the admitted session adapter, never by a URL.
class MediaBearer {
  final String value;
  final DateTime expiresAt;
  const MediaBearer(this.value, this.expiresAt);
}

class MediaAccessEnded implements Exception {
  const MediaAccessEnded();
}

/// Format is part of exact-URI admission, never inferred from a suffix.
enum AdmittedMediaFormat {
  mp3('audio/mpeg', 'mp3'),
  m4a('audio/mp4', 'm4a'),
  mp4('video/mp4', 'mp4'),
  png('image/png', 'png'),
  jpeg('image/jpeg', 'jpg'),
  webp('image/webp', 'webp');

  final String mimeType;
  final String extension;
  const AdmittedMediaFormat(this.mimeType, this.extension);
}

/// One session/workspace scope. Recreate after authority changes.
/// Exact URI bindings must come from an admitted manifest contract. An origin
/// alone is insufficient: it could include unrelated authenticated endpoints.
class AuthorizedMediaTransport {
  static const maxRangeBytes = 1024 * 1024;
  final Map<String, AdmittedMediaFormat> _bindings;
  final HttpClient Function() _clientFactory;
  final Future<MediaBearer> Function() bearer;
  final DateTime Function() now;
  final Duration deadline;
  final Set<HttpClient> _active = {};
  final Set<void Function()> _aborters = {};
  final _accessEnded = StreamController<void>.broadcast(sync: true);
  Stream<void> get accessEnded => _accessEnded.stream;
  bool _revoked = false;
  int _generation = 0;

  AuthorizedMediaTransport({
    required Map<Uri, AdmittedMediaFormat> admittedAssets,
    HttpClient Function()? clientFactory,
    required this.bearer,
    DateTime Function()? clock,
    this.deadline = const Duration(seconds: 20),
  }) : now = clock ?? DateTime.now,
       _clientFactory = clientFactory ?? HttpClient.new,
       _bindings = Map.unmodifiable({
         for (final entry in admittedAssets.entries)
           _binding(entry.key): entry.value,
       }) {
    if (deadline <= Duration.zero) {
      throw ArgumentError('Invalid deadline');
    }
  }

  static String _binding(Uri uri) {
    if (uri.scheme != 'https' ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        uri.path.isEmpty ||
        uri.path == '/' ||
        uri.pathSegments.any(
          (part) =>
              part == '.' ||
              part == '..' ||
              part.contains('/') ||
              part.contains('\\'),
        )) {
      throw const FormatException('Invalid admitted media URI');
    }
    return uri.toString();
  }

  void revoke() {
    if (_revoked) {
      return;
    }
    _revoked = true;
    _accessEnded.add(null);
    cancelRequests();
  }

  void cancelRequests() {
    _generation++;
    for (final abort in _aborters.toList()) {
      abort();
    }
    for (final client in _active.toList()) {
      client.close(force: true);
    }
  }

  void _check(int generation, DateTime expiry) {
    if (_revoked || generation != _generation || !now().isBefore(expiry)) {
      throw const MediaAccessEnded();
    }
  }

  Future<MediaResponse> head(Uri uri) => _request(uri, null, null);

  Future<MediaResponse> range(Uri uri, int first, int last) {
    if (first < 0 || last < first || last - first + 1 > maxRangeBytes) {
      throw ArgumentError('A closed range of at most 1 MiB is required');
    }
    return _request(uri, first, last);
  }

  Future<MediaResponse> _request(Uri uri, int? first, int? last) async {
    if (_revoked || !_bindings.containsKey(_binding(uri))) {
      throw const MediaAccessEnded();
    }
    final generation = _generation;
    final credential = await bearer().timeout(
      deadline,
    ); // Resolve afresh for every request.
    _check(generation, credential.expiresAt);
    if (credential.value.isEmpty ||
        credential.value.contains(RegExp(r'[\s]'))) {
      throw const MediaAccessEnded();
    }
    final format = _bindings[_binding(uri)]!;
    final client = _clientFactory()
      ..autoUncompress = false
      ..connectionTimeout = deadline;
    _active.add(client);
    final lifetime = credential.expiresAt.difference(now());
    final timeout = lifetime < deadline ? lifetime : deadline;
    final interrupted = Completer<MediaResponse>();
    void Function()? abortBody;
    void abort() {
      if (!interrupted.isCompleted) {
        interrupted.completeError(const MediaAccessEnded());
      }
      abortBody?.call();
      client.close(force: true);
    }

    _aborters.add(abort);
    final timer = Timer(timeout, abort);
    StreamSubscription<List<int>>? bodySubscription;
    try {
      final operation = (() async {
        final request = await client.openUrl(
          first == null ? 'HEAD' : 'GET',
          uri,
        );
        request.followRedirects = false;
        request.headers.set(
          HttpHeaders.authorizationHeader,
          'Bearer ${credential.value}',
        );
        request.headers.set(HttpHeaders.acceptEncodingHeader, 'identity');
        if (first != null) {
          request.headers.set(HttpHeaders.rangeHeader, 'bytes=$first-$last');
        }
        final response = await request.close();
        if (interrupted.isCompleted) {
          throw const MediaAccessEnded();
        }
        _check(generation, credential.expiresAt);
        if (response.statusCode == 401 || response.statusCode == 403) {
          revoke(); // Invalidate only this admitted media scope.
          throw const MediaAccessEnded();
        }
        if (response.statusCode != (first == null ? 200 : 206) ||
            (response.headers.value(HttpHeaders.contentEncodingHeader) ??
                    'identity') !=
                'identity') {
          throw const FormatException('Media response rejected');
        }
        final rawType = response.headers.value(HttpHeaders.contentTypeHeader);
        if (rawType == null ||
            ContentType.parse(rawType).mimeType.toLowerCase() !=
                format.mimeType) {
          throw const FormatException('Media type does not match admission');
        }
        final etag = response.headers.value(HttpHeaders.etagHeader);
        if (etag == null || !RegExp(r'^"[^"\r\n]+"$').hasMatch(etag)) {
          throw const FormatException('Immutable media validator required');
        }
        final length = response.contentLength;
        if (length < 0 || (first != null && length != last! - first + 1)) {
          throw const FormatException('Invalid media length');
        }
        int total = length;
        if (first != null) {
          final match = RegExp(r'^bytes (\d+)-(\d+)/(\d+)$').firstMatch(
            response.headers.value(HttpHeaders.contentRangeHeader) ?? '',
          );
          if (match == null ||
              int.parse(match[1]!) != first ||
              int.parse(match[2]!) != last ||
              int.parse(match[3]!) <= last!) {
            throw const FormatException('Invalid media range');
          }
          total = int.parse(match[3]!);
        }
        final bytes = BytesBuilder(copy: false);
        final bodyDone = Completer<void>();
        abortBody = () {
          if (!bodyDone.isCompleted) {
            bodyDone.completeError(const MediaAccessEnded());
          }
        };
        bodySubscription = response.listen(
          (chunk) {
            if (bodyDone.isCompleted) {
              return;
            }
            try {
              _check(generation, credential.expiresAt);
              if (first == null || bytes.length + chunk.length > length) {
                throw const FormatException('Unexpected media body');
              }
              bytes.add(chunk);
            } catch (error, stack) {
              bodyDone.completeError(error, stack);
            }
          },
          onError: (Object error, StackTrace stack) {
            if (!bodyDone.isCompleted) {
              bodyDone.completeError(error, stack);
            }
          },
          onDone: () {
            if (!bodyDone.isCompleted) {
              bodyDone.complete();
            }
          },
        );
        await bodyDone.future;
        _check(generation, credential.expiresAt);
        if (first != null && bytes.length != length) {
          throw const FormatException('Truncated media body');
        }
        return MediaResponse(
          total,
          bytes.takeBytes(),
          credential.expiresAt,
          etag,
          format,
        );
      })();
      return await Future.any([operation, interrupted.future]);
    } finally {
      timer.cancel();
      _aborters.remove(abort);
      await bodySubscription?.cancel();
      _active.remove(client);
      client.close(force: true);
    }
  }
}

class MediaResponse {
  final int totalBytes;
  final Uint8List bytes;
  final DateTime expiresAt;
  final String etag;
  final AdmittedMediaFormat format;
  const MediaResponse(
    this.totalBytes,
    this.bytes,
    this.expiresAt,
    this.etag,
    this.format,
  );
}
