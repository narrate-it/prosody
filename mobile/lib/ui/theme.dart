import 'dart:math' as math;
import 'package:flutter/material.dart';

const ink = Color(0xFF161A24),
    paper = Color(0xFFF5F6F8),
    blue = Color(0xFF3154D9),
    muted = Color(0xFFA8B2C4);
ThemeData prosodyTheme(Brightness brightness) {
  final dark = brightness == Brightness.dark;
  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    scaffoldBackgroundColor: dark ? ink : paper,
    colorScheme: ColorScheme.fromSeed(
      seedColor: blue,
      brightness: brightness,
      surface: dark ? ink : paper,
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: Colors.transparent,
      elevation: 0,
      centerTitle: false,
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: blue,
        foregroundColor: Colors.white,
        minimumSize: const Size(48, 52),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
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
    child: CustomPaint(size: Size.square(size), painter: _MarkPainter()),
  );
}

class _MarkPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final p = Paint()
      ..color = blue
      ..style = PaintingStyle.stroke
      ..strokeWidth = size.width * .13
      ..strokeCap = StrokeCap.round;
    final path = Path()
      ..moveTo(size.width * .28, size.height * .87)
      ..lineTo(size.width * .28, size.height * .39)
      ..cubicTo(
        size.width * .28,
        size.height * .07,
        size.width * .85,
        size.height * .07,
        size.width * .82,
        size.height * .43,
      )
      ..cubicTo(
        size.width * .80,
        size.height * .68,
        size.width * .43,
        size.height * .72,
        size.width * .43,
        size.height * .43,
      );
    canvas.drawPath(path, p);
  }

  @override
  bool shouldRepaint(_MarkPainter old) => false;
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
          painter: _WavePainter(_motion.value, widget.playing),
          size: const Size(double.infinity, 220),
        ),
      ),
    ),
  );
}

class _WavePainter extends CustomPainter {
  final double phase;
  final bool playing;
  _WavePainter(this.phase, this.playing);
  @override
  void paint(Canvas canvas, Size s) {
    final paint = Paint()
      ..color = blue
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
      old.phase != phase || old.playing != playing;
}
