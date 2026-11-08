import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/ui/theme.dart';

void main() {
  for (final brightness in Brightness.values) {
    testWidgets('Brand surfaces and controls: ${brightness.name}', (
      tester,
    ) async {
      final theme = prosodyTheme(brightness);
      final scheme = theme.colorScheme;
      expect(scheme.primary, brightness == Brightness.dark ? paper : ink);
      expect(
        theme.scaffoldBackgroundColor,
        brightness == Brightness.dark ? ink : fog,
      );
      expect(_contrast(scheme.onSurface, scheme.surface), greaterThan(4.5));
      expect(
        _contrast(scheme.onSurfaceVariant, scheme.surface),
        greaterThan(4.5),
      );
      expect(_contrast(scheme.onPrimary, scheme.primary), greaterThan(4.5));
      await tester.binding.setSurfaceSize(const Size(390, 1000));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      await tester.pumpWidget(
        MaterialApp(
          theme: theme,
          home: Scaffold(
            appBar: AppBar(title: const Text('Prosody')),
            body: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const ProsodyMark(size: 80),
                  const SizedBox(height: 24),
                  Text(
                    'Your agents.\nYour perspective.',
                    style: theme.textTheme.headlineLarge,
                  ),
                  const SizedBox(height: 24),
                  const Card(
                    child: ListTile(
                      leading: Icon(Icons.headphones),
                      title: Text('Morning briefing'),
                      subtitle: Text('Ready to listen'),
                    ),
                  ),
                  const SizedBox(height: 16),
                  const TextField(
                    decoration: InputDecoration(labelText: 'Feed name'),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: () {},
                    child: const Text('Listen now'),
                  ),
                  TextButton(
                    onPressed: () {},
                    child: const Text('View activity'),
                  ),
                  const ListeningWave(playing: false),
                ],
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      await expectLater(
        find.byType(Scaffold),
        matchesGoldenFile('goldens/brand-${brightness.name}.png'),
      );
    });
  }
}

double _contrast(Color a, Color b) {
  final x = a.computeLuminance(), y = b.computeLuminance();
  return x > y ? (x + .05) / (y + .05) : (y + .05) / (x + .05);
}
