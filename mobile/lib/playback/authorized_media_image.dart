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

  void _load() {
    final generation = ++_generation;
    final buffer = widget.buffer;
    _ended = buffer.accessEnded.listen((_) {
      _generation++;
      _clear(buffer);
      if (mounted) {
        setState(() => _failed = true);
      }
    });
    unawaited(() async {
      try {
        final file = await buffer.prepare(widget.uri);
        if (!mounted || generation != _generation || buffer.isClosed) {
          await buffer.release(file);
          return;
        }
        setState(() => _file = file);
      } catch (_) {
        if (mounted && generation == _generation) {
          setState(() => _failed = true);
        }
      }
    }());
  }

  void _clear(AuthorizedMediaBuffer buffer) {
    final file = _file;
    _file = null;
    if (file != null) {
      unawaited(() async {
        await FileImage(file).evict();
        await buffer.release(file);
      }());
    }
  }

  @override
  void didUpdateWidget(covariant AuthorizedMediaImage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.uri != oldWidget.uri || widget.buffer != oldWidget.buffer) {
      _generation++;
      unawaited(_ended?.cancel());
      _clear(oldWidget.buffer);
      _failed = false;
      _load();
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
    unawaited(_ended?.cancel());
    _clear(widget.buffer);
    super.dispose();
  }
}
