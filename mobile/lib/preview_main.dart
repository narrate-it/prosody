// Explicit debug preview entry point. Production main.dart never imports this.
import 'dart:async';
import 'dart:convert';
import 'dart:developer' as developer;
import 'dart:io';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:path_provider/path_provider.dart';

import 'playback/handler.dart';
import 'data/models.dart';
import 'preview/preview_api.dart';
import 'preview/local_media.dart';
import 'preview/live_provider.dart';
import 'preview/live_session.dart';
import 'preview/live_screen.dart';
import 'ui/feed.dart';
import 'ui/theme.dart';

class _PreviewAudio extends audio.AudioPlayer {
  final Map<String, String> files;
  _PreviewAudio(this.files);
  @override
  Future<Duration?> setUrl(
    String url, {
    Map<String, String>? headers,
    Duration? initialPosition,
    bool preload = true,
    dynamic tag,
  }) {
    final paths = {
      for (final id in ['welcome', 'design', 'next'])
        'https://preview.invalid/$id.mp3': 'assets/preview/$id.mp3',
    };
    final asset = paths[url];
    if (asset != null) {
      return setAsset(
        asset,
        initialPosition: initialPosition,
        preload: preload,
        tag: tag,
      );
    }
    final file = files[url];
    if (file == null) throw StateError('Preview media must be local');
    return setFilePath(
      file,
      initialPosition: initialPosition,
      preload: preload,
      tag: tag,
    );
  }
}

Future<void> main() async {
  if (!kDebugMode) throw StateError('Preview is debug-only');
  WidgetsFlutterBinding.ensureInitialized();
  final support = await getApplicationSupportDirectory();
  final clips = Directory('${support.path}/prosody-preview-clips');
  await clips.create(recursive: true);
  final live = LiveSession(
    ledger: File('${support.path}/prosody-live-budget.json'),
  );
  await live.initialize();
  // Provision only via the authenticated, attached debug VM service. The key
  // remains in memory; it is never an asset, build flag, preference or log value.
  developer.registerExtension('ext.prosody.configureLivePreview', (
    method,
    args,
  ) async {
    try {
      final key = args['key'];
      if (key == null || args['sessionId'] != 'prosody-live-20261101-1') {
        return developer.ServiceExtensionResponse.error(
          -32602,
          'Invalid preview configuration',
        );
      }
      live.configure(key);
      return developer.ServiceExtensionResponse.result(
        jsonEncode({
          'configured': live.configured,
          'remainingAttempts': live.remainingAttempts,
        }),
      );
    } catch (_) {
      return developer.ServiceExtensionResponse.error(
        -32602,
        'Preview configuration unavailable',
      );
    }
  });
  final durations = (jsonDecode(
    await rootBundle.loadString('assets/preview/durations.json'),
  ) as Map<String, dynamic>).cast<String, int>();
  final files = <String, String>{};
  final playback = await AudioService.init<ProsodyPlayback>(
    builder: () => ProsodyPlayback(audioPlayer: _PreviewAudio(files)),
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'run.prosody.preview.playback',
      androidNotificationChannelName: 'Prosody preview',
      androidNotificationOngoing: true,
    ),
  );
  await playback.configure();
  runApp(
    ProsodyPreview(
      playback: playback,
      api: PreviewApi(
        durationsMs: durations,
        onDelete: (id) => deletePreviewMedia(files, id),
      ),
      live: live,
      clips: clips,
      files: files,
    ),
  );
}

class ProsodyPreview extends StatefulWidget {
  final ProsodyPlayback playback;
  final PreviewApi api;
  final LiveSession live;
  final Directory clips;
  final Map<String, String> files;
  const ProsodyPreview({
    super.key,
    required this.playback,
    required this.api,
    required this.live,
    required this.clips,
    required this.files,
  });
  @override
  State<ProsodyPreview> createState() => _ProsodyPreviewState();
}

class _ProsodyPreviewState extends State<ProsodyPreview> {
  final _navigator = GlobalKey<NavigatorState>();
  bool _showFeed = true;
  int _feedRevision = 0;
  Future<void> _createBriefing() async {
    await widget.playback.pause();
    await _navigator.currentState!.push<void>(
      MaterialPageRoute<void>(
        builder: (_) => LiveBriefingScreen(
          session: widget.live,
          onGenerated: _publishLocal,
        ),
      ),
    );
  }

  Future<void> _deleteBriefing(Briefing item) async {
    // Preview-only personal destination. Production requires qualified authority.
    final personal = await widget.api.items('personal');
    if (!mounted ||
        !_showFeed ||
        !personal.items.any((value) => value.id == item.id)) {
      throw StateError('Briefing is no longer available in this inbox');
    }
    if (widget.playback.current?.id == item.id) await widget.playback.stop();
    if (!mounted || !_showFeed) throw StateError('Preview ended');
    try {
      await widget.api.deleteItem(item.id);
    } finally {
      // Restore a usable queue after a failed cleanup as well as success.
      if (mounted && _showFeed) setState(() => _feedRevision++);
    }
  }

  Future<void> _publishLocal(LiveBriefing result) async {
    void requireCurrentSession() {
      if (!mounted || !widget.live.canPublishResult(result)) {
        throw StateError('Preview session ended');
      }
    }

    requireCurrentSession();
    if (result.mp3.isEmpty || result.mp3.length > 8 * 1024 * 1024) {
      throw StateError('Generated audio is outside preview limits');
    }
    final id = 'live-${DateTime.now().microsecondsSinceEpoch}';
    final file = File('${widget.clips.path}/$id.mp3');
    final probe = audio.AudioPlayer();
    var accepted = false;
    try {
      await file.writeAsBytes(result.mp3, flush: true);
      final duration = await probe
          .setFilePath(file.path)
          .timeout(const Duration(seconds: 20));
      if (duration == null ||
          duration <= Duration.zero ||
          duration > const Duration(minutes: 10)) {
        throw StateError('Generated audio duration is outside preview limits');
      }
      requireCurrentSession();
      await widget.playback.stop();
      requireCurrentSession();
      widget.api.addLiveBriefing(
        id: id,
        source: result.source,
        script: result.script,
        durationMs: duration.inMilliseconds,
      );
      widget.files['https://preview.invalid/$id.mp3'] = file.path;
      accepted = true;
      setState(() => _feedRevision++);
    } finally {
      await probe.dispose();
      if (!accepted && await file.exists()) await file.delete();
    }
  }

  @override
  void dispose() {
    widget.api.close();
    widget.live.dispose();
    unawaited(widget.playback.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
    navigatorKey: _navigator,
    title: 'Prosody preview',
    debugShowCheckedModeBanner: false,
    theme: prosodyTheme(Brightness.light),
    darkTheme: prosodyTheme(Brightness.dark),
    builder: (context, child) => ListenableBuilder(
      listenable: widget.live,
      builder: (context, _) => Banner(
        message: widget.live.configured ? 'LIVE PREVIEW' : 'OFFLINE DEMO',
        location: BannerLocation.topEnd,
        color: blue,
        child: child!,
      ),
    ),
    home: _showFeed
        ? FeedScreen(
            key: ValueKey(_feedRevision),
            api: widget.api,
            playback: widget.playback,
            onCreateBriefing: _createBriefing,
            onDeleteBriefing: _deleteBriefing,
            onSignOut: () {
              widget.live.clearKey();
              setState(() => _showFeed = false);
            },
          )
        : Scaffold(
            body: SafeArea(
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const ProsodyMark(size: 80),
                    const SizedBox(height: 24),
                    const Text(
                      'Sample briefings stay local. Live generation requires an explicitly configured test session. Progress is local to this preview.',
                    ),
                    const SizedBox(height: 24),
                    FilledButton(
                      onPressed: () => setState(() => _showFeed = true),
                      child: const Text('Open preview feed'),
                    ),
                  ],
                ),
              ),
            ),
          ),
  );
}
