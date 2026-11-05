import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';

import 'authorized_media_buffer.dart';

/// Opt-in image handoff. The owner provides admitted format metadata and a
/// session-scoped buffer; no network ImageProvider ever receives credentials.
class AuthorizedMediaImage extends StatefulWidget {
  final AuthorizedMediaBuffer buffer;
  final Uri uri;
  final BoxFit fit;
  const AuthorizedMediaImage({
    super.key,
    required this.buffer,
    required this.uri,
    this.fit = BoxFit.contain,
  });

  @override
  State<AuthorizedMediaImage> createState() => _AuthorizedMediaImageState();
}

class _AuthorizedMediaImageState extends State<AuthorizedMediaImage> {
  StreamSubscription<void>? _ended;
  File? _file;
  bool _failed = false;
  int _generation = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _work = Future.value();
  final Map<File, AuthorizedMediaBuffer> _retired = {};

  void _enqueue(Future<void> Function() operation) {
    _work = _work.then((_) => operation()).catchError((Object error) {
      if (mounted) {
        setState(() => _failed = true);
      }
    });
  }

  Future<void> _release(AuthorizedMediaBuffer buffer, File? file) async {
    if (file == null) {
      return;
    }
    _retired[file] = buffer;
    try {
      await FileImage(file).evict();
    } finally {
      await buffer.release(file);
      _retired.remove(file); // Only after confirmed filesystem release.
    }
  }

  Future<void> _retryRetired() async {
    var failures = 0;
    for (final entry in _retired.entries.toList()) {
      try {
        await _release(entry.value, entry.key);
      } catch (_) {
        failures++;
      }
    }
    if (failures != 0) {
      throw MediaBufferCleanupFailed(failures);
    }
  }

  void _load({AuthorizedMediaBuffer? previousBuffer}) {
    final generation = ++_generation;
    final buffer = widget.buffer;
    final uri = widget.uri;
    final previousFile = _file;
    _file = null;
    if (previousFile != null) {
      _retired[previousFile] = previousBuffer ?? buffer;
    }
    final previousCancellation = _ended?.cancel();
    _ended = buffer.accessEnded.listen((_) {
      _generation++;
      final file = _file;
      _file = null;
      if (file != null) {
        _retired[file] = buffer;
      }
      _enqueue(_retryRetired);
      if (mounted) {
        setState(() => _failed = true);
      }
    });
    // Per-widget serial work: an obsolete download must settle and release its
    // reservation before its replacement starts. Never cancel a shared buffer.
    _enqueue(() async {
      await previousCancellation;
      await _retryRetired();
      if (!mounted || generation != _generation || buffer.isClosed) {
        return;
      }
      try {
        final file = await buffer.prepare(uri);
        if (!mounted || generation != _generation || buffer.isClosed) {
          await _release(buffer, file);
          return;
        }
        setState(() => _file = file);
      } catch (_) {
        if (mounted && generation == _generation) {
          setState(() => _failed = true);
        }
      }
    });
  }

  @override
  void didUpdateWidget(covariant AuthorizedMediaImage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.uri != oldWidget.uri || widget.buffer != oldWidget.buffer) {
      _failed = false;
      _load(previousBuffer: oldWidget.buffer);
    }
  }

  @override
  Widget build(BuildContext context) => _failed
      ? const Center(child: Icon(Icons.broken_image_outlined))
      : _file == null
      ? const Center(child: CircularProgressIndicator())
      : Image.file(
          _file!,
          fit: widget.fit,
          errorBuilder: (_, _, _) =>
              const Center(child: Icon(Icons.broken_image_outlined)),
        );

  @override
  void dispose() {
    _generation++;
    final file = _file;
    _file = null;
    final subscription = _ended;
    final buffer = widget.buffer;
    if (file != null) {
      _retired[file] = buffer;
    }
    _enqueue(() async {
      await subscription?.cancel();
      await _retryRetired();
    });
    super.dispose();
  }
}
