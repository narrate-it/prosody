// Explicit local demo entry point. Production main.dart never imports this.
import 'dart:async';
import 'dart:convert';
import 'package:audio_service/audio_service.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'playback/handler.dart';
import 'preview/preview_api.dart';
import 'ui/feed.dart';
import 'ui/theme.dart';

class _BundledAudio extends audio.AudioPlayer {
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
    if (asset == null) throw StateError('Preview media must be bundled');
    return setAsset(
      asset,
      initialPosition: initialPosition,
      preload: preload,
      tag: tag,
    );
  }
}

Future<void> main() async {
  if (!kDebugMode) throw StateError('Offline preview is debug-only');
  WidgetsFlutterBinding.ensureInitialized();
  final durations =
      (jsonDecode(await rootBundle.loadString('assets/preview/durations.json'))
              as Map<String, dynamic>)
          .cast<String, int>();
  final playback = await AudioService.init<ProsodyPlayback>(
    builder: () => ProsodyPlayback(audioPlayer: _BundledAudio()),
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'run.prosody.preview.playback',
      androidNotificationChannelName: 'Prosody offline preview',
      androidNotificationOngoing: true,
    ),
  );
  await playback.configure();
  runApp(
    OfflinePreview(
      playback: playback,
      api: PreviewApi(durationsMs: durations),
    ),
  );
}

class OfflinePreview extends StatefulWidget {
  final ProsodyPlayback playback;
  final PreviewApi api;
  const OfflinePreview({super.key, required this.playback, required this.api});
  @override
  State<OfflinePreview> createState() => _OfflinePreviewState();
}

class _OfflinePreviewState extends State<OfflinePreview> {
  bool _showFeed = true;
  @override
  void dispose() {
    widget.api.close();
    unawaited(widget.playback.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Prosody offline preview',
    debugShowCheckedModeBanner: false,
    theme: prosodyTheme(Brightness.light),
    darkTheme: prosodyTheme(Brightness.dark),
    builder: (context, child) => Banner(
      message: 'OFFLINE DEMO',
      location: BannerLocation.topEnd,
      color: blue,
      child: child!,
    ),
    home: _showFeed
        ? FeedScreen(
            api: widget.api,
            playback: widget.playback,
            onSignOut: () => setState(() => _showFeed = false),
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
                      'Try Prosody with sample briefings. No account is needed. Changes last only for this preview session.',
                    ),
                    const SizedBox(height: 24),
                    FilledButton(
                      onPressed: () => setState(() => _showFeed = true),
                      child: const Text('Open sample feed'),
                    ),
                  ],
                ),
              ),
            ),
          ),
  );
}
