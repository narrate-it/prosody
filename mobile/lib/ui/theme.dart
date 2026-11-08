import 'dart:math' as math;

import 'package:flutter/material.dart';

// Owner-supplied Prosody black & white brand palette.
const ink = Color(0xFF0A0A0A),
    paper = Color(0xFFFFFFFF),
    fog = Color(0xFFF5F6F8),
    graphite = Color(0xFF5C5C62),
    signal = Color(0xFFFF4F1A);

ThemeData prosodyTheme(Brightness brightness) {
  final dark = brightness == Brightness.dark;
  final foreground = dark ? paper : ink;
  final background = dark ? ink : fog;
  final surface = dark ? const Color(0xFF1C1C1E) : paper;
  final secondaryText = dark ? const Color(0xFFB9B9BF) : graphite;
  final scheme =
      ColorScheme.fromSeed(
        seedColor: ink,
        brightness: brightness,
        dynamicSchemeVariant: DynamicSchemeVariant.monochrome,
      ).copyWith(
        primary: foreground,
        onPrimary: dark ? ink : paper,
        primaryContainer: dark ? graphite : fog,
        onPrimaryContainer: foreground,
        secondary: foreground,
        onSecondary: dark ? ink : paper,
        secondaryContainer: dark ? graphite : fog,
        onSecondaryContainer: foreground,
        tertiary: foreground,
        onTertiary: dark ? ink : paper,
        tertiaryContainer: dark ? graphite : fog,
        onTertiaryContainer: foreground,
        surface: surface,
        onSurface: foreground,
        onSurfaceVariant: secondaryText,
        surfaceContainerLowest: background,
        surfaceContainerLow: surface,
        surfaceContainer: surface,
        surfaceContainerHigh: dark ? const Color(0xFF29292C) : fog,
        surfaceContainerHighest: dark ? const Color(0xFF353538) : fog,
        outline: secondaryText,
        outlineVariant: dark ? graphite : const Color(0xFFD6D6DA),
        // Errors remain distinct from the brand's unread-only orange signal.
        error: dark ? const Color(0xFFFFB4AB) : const Color(0xFFBA1A1A),
        onError: dark ? const Color(0xFF690005) : paper,
      );
  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    scaffoldBackgroundColor: background,
    colorScheme: scheme,
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.transparent,
      elevation: 0,
      centerTitle: false,
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: foreground,
        foregroundColor: scheme.onPrimary,
        minimumSize: const Size(48, 52),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: scheme.surfaceContainerHighest,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: BorderSide.none,
      ),
    ),
    textTheme: const TextTheme(
      headlineLarge: TextStyle(
        fontSize: 38,
        height: 1.15,
        fontWeight: FontWeight.w600,
        letterSpacing: -1.3,
      ),
      headlineMedium: TextStyle(
        fontSize: 28,
        fontWeight: FontWeight.w600,
        letterSpacing: -.7,
      ),
    ),
  );
}

class ProsodyMark extends StatelessWidget {
  final double size;
  const ProsodyMark({super.key, this.size = 40});
  @override
  Widget build(BuildContext context) => Semantics(
    label: 'Prosody',
    child: CustomPaint(
      size: Size.square(size),
      painter: _MarkPainter(Theme.of(context).colorScheme.onSurface),
    ),
  );
}

class _MarkPainter extends CustomPainter {
  final Color color;
  _MarkPainter(this.color);

  @override
  void paint(Canvas canvas, Size size) {
    canvas.save();
    canvas.scale(size.width / 100, size.height / 100);
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 12
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    // Exact owner-supplied prosody-mark.svg path, in its 100x100 viewBox.
    final path = Path()
      ..moveTo(31, 85)
      ..cubicTo(37, 67, 45, 47, 53, 29)
      ..cubicTo(57, 19, 65, 13, 73, 15)
      ..cubicTo(83, 18, 87, 29, 83, 39)
      ..cubicTo(79, 49, 67, 55, 53, 53)
      ..cubicTo(41, 52, 29, 49, 17, 51);
    canvas.drawPath(path, paint);
    canvas.restore();
  }

  @override
  bool shouldRepaint(_MarkPainter old) => old.color != color;
}

// Decorative playback activity, deliberately not represented as an audio meter.
class ListeningWave extends StatefulWidget {
  final bool playing;
  const ListeningWave({super.key, required this.playing});
  @override
  State<ListeningWave> createState() => _ListeningWaveState();
}

class _ListeningWaveState extends State<ListeningWave>
    with SingleTickerProviderStateMixin {
  late final AnimationController _motion = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 4),
  );
  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _update();
  }

  @override
  void didUpdateWidget(ListeningWave old) {
    super.didUpdateWidget(old);
    _update();
  }

  void _update() {
    if (widget.playing && !MediaQuery.disableAnimationsOf(context)) {
      _motion.repeat();
    } else {
      _motion.stop();
    }
  }

  @override
  void dispose() {
    _motion.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: SizedBox(
      height: 220,
      child: AnimatedBuilder(
        animation: _motion,
        builder: (context, _) => CustomPaint(
          painter: _WavePainter(
            _motion.value,
            widget.playing,
            Theme.of(context).colorScheme.primary,
          ),
          size: const Size(double.infinity, 220),
        ),
      ),
    ),
  );
}

class _WavePainter extends CustomPainter {
  final double phase;
  final bool playing;
  final Color color;
  _WavePainter(this.phase, this.playing, this.color);
  @override
  void paint(Canvas canvas, Size s) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = 4
      ..strokeCap = StrokeCap.round;
    for (int i = 0; i < 43; i++) {
      final x = s.width * .1 + i * s.width * .8 / 42;
      final envelope = math.sin(math.pi * i / 42);
      final motion = playing
          ? .45 + .55 * (math.sin(i * .62 + phase * math.pi * 2).abs())
          : .35;
      final h = 8 + envelope * 125 * motion;
      canvas.drawLine(
        Offset(x, s.height / 2 - h / 2),
        Offset(x, s.height / 2 + h / 2),
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(_WavePainter old) =>
      old.phase != phase || old.playing != playing || old.color != color;
}
