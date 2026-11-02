import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/preview/live_provider.dart';
import 'package:prosody/preview/live_screen.dart';
import 'package:prosody/preview/live_session.dart';

class _DelayedProvider implements LiveBriefingProvider {
  final reserved = Completer<void>();
  final result = Completer<LiveBriefing>();
  int calls = 0;
  @override
  Future<LiveBriefing> generate({
    required String key,
    required String source,
    required Future<void> Function() reserveAttempt,
  }) async {
    calls++;
    await reserveAttempt();
    reserved.complete();
    return result.future;
  }

  @override
  void close() {}
}

void main() {
  for (final leaveEditor in [true, false]) {
    testWidgets(
      'reserved result survives navigation without popping next route (removed: $leaveEditor)',
      (tester) async {
        final directory = Directory.systemTemp.createTempSync(
          'prosody-live-screen-',
        );
        final provider = (await tester.runAsync(
          () async => _DelayedProvider(),
        ))!;
        final session = LiveSession(
          ledger: File('${directory.path}/budget'),
          client: provider,
        );
        final navigator = GlobalKey<NavigatorState>();
        final published = <LiveBriefing>[];
        final delivered = (await tester.runAsync(
          () async => Completer<void>(),
        ))!;
        try {
          await tester.runAsync(() async {
            await session.initialize();
            session.configure('synthetic-key');
          });
          await tester.pumpWidget(
            MaterialApp(
              navigatorKey: navigator,
              home: const Scaffold(body: Text('Feed')),
            ),
          );
          unawaited(
            navigator.currentState!.push<void>(
              MaterialPageRoute<void>(
                builder: (_) => LiveBriefingScreen(
                  session: session,
                  onGenerated: (briefing) async {
                    published.add(briefing);
                    delivered.complete();
                  },
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          await tester.enterText(find.byType(TextField), 'A source update.');
          await tester.ensureVisible(find.text('Generate briefing'));
          await tester.pumpAndSettle();
          await tester.runAsync(() async {
            await tester.tap(find.text('Generate briefing'));
            await provider.reserved.future.timeout(const Duration(seconds: 5));
          });
          await tester.pump();
          expect(session.remainingAttempts, 3);
          expect(
            tester.widget<FilledButton>(find.byType(FilledButton)).onPressed,
            isNull,
          );
          if (leaveEditor) {
            await tester.pageBack();
            await tester.pumpAndSettle();
            expect(find.text('Feed'), findsOneWidget);
          }
          unawaited(
            navigator.currentState!.push<void>(
              MaterialPageRoute<void>(
                builder: (_) => const Scaffold(body: Text('Another screen')),
              ),
            ),
          );
          await tester.pumpAndSettle();
          final briefing = LiveBriefing(
            source: 'A source update.',
            script: 'Spoken update.',
            mp3: Uint8List.fromList([1, 2, 3]),
          );
          await tester.runAsync(() async {
            provider.result.complete(briefing);
            await delivered.future.timeout(const Duration(seconds: 5));
          });
          await tester.pumpAndSettle();
          expect(published, [same(briefing)]);
          expect(provider.calls, 1);
          expect(session.remainingAttempts, 3);
          expect(session.busy, isFalse);
          expect(find.text('Another screen'), findsOneWidget);
          expect(tester.takeException(), isNull);
        } finally {
          await tester.pumpWidget(const SizedBox.shrink());
          session.dispose();
          directory.deleteSync(recursive: true);
        }
      },
    );
  }
}
