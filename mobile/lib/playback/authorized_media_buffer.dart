import 'dart:async';
import 'dart:io';

import '../data/authorized_media.dart';

/// Ephemeral decoder handoff, not offline access or an authority source.
/// The owner must stop/dispose decoders on [onAccessEnded] before deletion.
class AuthorizedMediaBuffer {
  final AuthorizedMediaTransport transport;
  final int maxAssetBytes;
  final Future<void> Function() onAccessEnded;
  Timer? _expiry;
  bool _closed = false;
  final MediaBufferStorage storage;
  int _serial = 0;
  int _generation = 0;

  void cancelPending() {
    _generation++;
    transport.cancelRequests();
  }

  int _reservedBytes = 0;
  final Map<String, int> _sizes = {};
  final Set<Future<void>> _pending = {};
  final _ended = StreamController<void>.broadcast(sync: true);
  Stream<void> get accessEnded => _ended.stream;
  bool get isClosed => _closed;
  Future<void>? _closing;
  StreamSubscription<void>? _transportEnded;
  Object? automaticCleanupFailure;

  void _automaticClose() {
    unawaited(
      close().catchError((Object error) {
        automaticCleanupFailure = error;
      }),
    );
  }

  AuthorizedMediaBuffer({
    required this.transport,
    required this.maxAssetBytes,
    required this.onAccessEnded,
    this.storage = const MediaBufferStorage(),
  }) {
    // Deliberately finite. Larger assets need a separately qualified streaming
    // transport; do not silently download an entire long supplied recording.
    if (maxAssetBytes <= 0 || maxAssetBytes > 64 * 1024 * 1024) {
      throw ArgumentError('Asset buffer limit must be within 64 MiB');
    }
    _transportEnded = transport.accessEnded.listen((_) => _automaticClose());
  }

  Future<File> prepare(Uri uri) {
    final completer = Completer<File>();
    late final Future<void> operation;
    operation = (() async {
      try {
        completer.complete(await _prepare(uri));
      } catch (error, stack) {
        completer.completeError(error, stack);
      } finally {
        _pending.remove(operation);
      }
    })();
    _pending.add(operation);
    return completer.future;
  }

  Future<File> _prepare(Uri uri) async {
    final generation = _generation;
    if (_closed) {
      throw const MediaAccessEnded();
    }
    final metadata = await transport.head(uri);
    if (_closed ||
        generation != _generation ||
        metadata.totalBytes <= 0 ||
        metadata.totalBytes > maxAssetBytes) {
      throw const MediaAccessEnded();
    }
    if (_reservedBytes + metadata.totalBytes > maxAssetBytes) {
      throw const MediaAccessEnded();
    }
    _reservedBytes += metadata.totalBytes;
    _armExpiry(metadata.expiresAt);
    late final Directory directory;
    try {
      directory = await storage.createDirectory();
    } catch (_) {
      _reservedBytes -= metadata.totalBytes;
      rethrow;
    }
    _directories.add(directory);
    _sizes[directory.path] = metadata.totalBytes;
    // Each preparation owns its directory. Only our generated names are used.
    final file = File(
      '${directory.path}/${_serial++}.${metadata.format.extension}',
    );
    IOSink? sink;
    try {
      if (_closed || generation != _generation) {
        throw const MediaAccessEnded();
      }
      sink = storage.openWrite(file);
      for (var first = 0; first < metadata.totalBytes;) {
        if (_closed || generation != _generation) {
          throw const MediaAccessEnded();
        }
        final last = (first + AuthorizedMediaTransport.maxRangeBytes - 1).clamp(
          first,
          metadata.totalBytes - 1,
        );
        final response = await transport.range(uri, first, last);
        if (_closed ||
            generation != _generation ||
            response.totalBytes != metadata.totalBytes ||
            response.etag != metadata.etag ||
            response.format != metadata.format) {
          throw const MediaAccessEnded();
        }
        _armExpiry(response.expiresAt);
        sink.add(response.bytes);
        await sink.flush();
        first = last + 1;
      }
      await sink.close();
      if (_closed || generation != _generation) {
        throw const MediaAccessEnded();
      }
      return file;
    } catch (error, stack) {
      // Closing a failed sink may fail again. Always attempt exact owned
      // deletion/accounting, and preserve the original write/transport failure.
      try {
        try {
          await sink?.close();
        } finally {
          await _deleteOwned(directory.path);
        }
      } catch (_) {
        // The primary failure remains authoritative. Failed deletion retains
        // ownership/accounting for a later close/release retry.
      }
      Error.throwWithStackTrace(error, stack);
    }
  }

  final Map<String, Future<void>> _deletions = {};
  Future<void> release(File file) {
    final path = file.parent.path;
    final existing = _deletions[path];
    if (existing != null) {
      return existing;
    }
    final operation = _deleteOwned(path).whenComplete(() {
      // remove returns this very operation. Returning/awaiting it would make
      // whenComplete wait for its own completion forever.
      unawaited(_deletions.remove(path));
    });
    _deletions[path] = operation;
    return operation;
  }

  Future<void> _deleteOwned(String path) async {
    final owned = _directories
        .where((directory) => directory.path == path)
        .firstOrNull;
    if (owned == null) {
      return;
    }
    await storage.deleteDirectory(owned);
    _directories.remove(owned);
    _reservedBytes -= _sizes.remove(path) ?? 0;
  }

  final Set<Directory> _directories = {};
  DateTime? _expiresAt;
  void _armExpiry(DateTime expiresAt) {
    if (_expiresAt != null && !_expiresAt!.isAfter(expiresAt)) {
      return;
    }
    _expiresAt = expiresAt;
    _expiry?.cancel();
    _expiry = Timer(expiresAt.difference(transport.now()), () {
      _automaticClose();
    });
  }

  Future<void> close() => _closing ??= _close();
  Future<void> _close() async {
    _closed = true;
    _ended.add(null);
    _expiry?.cancel();
    await _transportEnded?.cancel();
    transport.revoke();
    await onAccessEnded();
    await Future.wait(_pending.toList());
    for (final directory in _directories.toList()) {
      await release(File('${directory.path}/owned'));
    }
    await _ended.close();
  }
}

/// Narrow injectable filesystem seam. Tests can fail writes/flushes/closes
/// without global I/O overrides or changing production TLS behavior.
class MediaBufferStorage {
  const MediaBufferStorage();
  Future<Directory> createDirectory() =>
      Directory.systemTemp.createTemp('prosody-media-');
  IOSink openWrite(File file) => file.openWrite();
  Future<void> deleteDirectory(Directory directory) async {
    if (await directory.exists()) {
      await directory.delete(recursive: true);
    }
  }
}
