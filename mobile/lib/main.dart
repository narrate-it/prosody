import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';

import 'data/api.dart';
import 'data/session.dart';
import 'data/service_configuration.dart';
import 'playback/handler.dart';
import 'ui/feed.dart';
import 'ui/theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final playback = await AudioService.init<ProsodyPlayback>(
    builder: ProsodyPlayback.new,
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'run.prosody.playback',
      androidNotificationChannelName: 'Prosody playback',
      androidNotificationOngoing: true,
    ),
  );
  await playback.configure();
  ServiceConfiguration? configuration;
  try {
    configuration = ServiceConfiguration.parse(
      origin: const String.fromEnvironment('PROSODY_ORIGIN'),
      issuer: const String.fromEnvironment('PROSODY_ISSUER_ORIGIN'),
      audience: const String.fromEnvironment('PROSODY_RESOURCE_AUDIENCE'),
    );
  } on FormatException {
    // Invalid release inputs stay unavailable; never fall back to preview data.
  }
  runApp(
    ProsodyApp(
      playback: playback,
      origin: configuration?.origin,
      session: configuration?.createSession(),
    ),
  );
}

class ProsodyApp extends StatefulWidget {
  final InteractiveSession? session;
  final ProsodyPlayback playback;
  final Uri? origin;
  const ProsodyApp({
    super.key,
    this.session,
    required this.playback,
    this.origin,
  });
  @override
  State<ProsodyApp> createState() => _ProsodyAppState();
}

class _ProsodyAppState extends State<ProsodyApp> {
  ProsodyApi? _api;
  bool _signedIn = false, _busy = true;
  String? _error;
  @override
  void initState() {
    super.initState();
    unawaited(_restore());
  }

  Future<void> _restore() async {
    try {
      final session = widget.session;
      if (session == null) {
        if (mounted) {
          setState(() => _error = 'Sign-in is not available in this build.');
        }
        return;
      }
      if (widget.origin == null || !widget.origin!.hasScheme) {
        throw StateError('This build has no service configuration.');
      }
      _api = ProsodyApi(widget.origin!, session);
      final restored = await session.restore();
      if (mounted) setState(() => _signedIn = restored);
    } catch (_) {
      if (mounted) {
        setState(() => _error = 'This build cannot connect to Prosody yet.');
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _signIn() async {
    final session = widget.session;
    if (session == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await session.signIn();
      if (mounted) setState(() => _signedIn = true);
    } catch (_) {
      if (mounted) {
        setState(() => _error = 'Sign-in did not finish. Please try again.');
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  void dispose() {
    _api?.close();
    unawaited(widget.playback.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Prosody',
    debugShowCheckedModeBanner: false,
    theme: prosodyTheme(Brightness.light),
    darkTheme: prosodyTheme(Brightness.dark),
    home: _signedIn && _api != null
        ? FeedScreen(
            api: _api!,
            playback: widget.playback,
            onSignOut: () => setState(() => _signedIn = false),
          )
        : Scaffold(
            body: SafeArea(
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Builder(
                  builder: (context) => Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Spacer(),
                      const ProsodyMark(size: 80),
                      const SizedBox(height: 32),
                      Text(
                        'Less reading.\nMore perspective.',
                        style: Theme.of(context).textTheme.headlineLarge,
                      ),
                      const SizedBox(height: 20),
                      const Text(
                        'A quiet place for your agents’ briefings. Listen, watch, and carry on.',
                        style: TextStyle(fontSize: 18, height: 1.5),
                      ),
                      const Spacer(),
                      if (_error != null)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 20),
                          child: Text(_error!),
                        ),
                      SizedBox(
                        width: double.infinity,
                        child: FilledButton(
                          onPressed: _busy || _api == null ? null : _signIn,
                          child: Text(
                            _busy ? 'Connecting…' : 'Continue to Prosody',
                          ),
                        ),
                      ),
                      const SizedBox(height: 20),
                    ],
                  ),
                ),
              ),
            ),
          ),
  );
}
