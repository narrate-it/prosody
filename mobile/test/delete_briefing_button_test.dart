import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/ui/delete_briefing_button.dart';

Future<void> open(WidgetTester tester, Future<void> Function() action) async {
  await tester.pumpWidget(
    MaterialApp(
      home: Builder(
        builder: (context) => Scaffold(
          body: TextButton(
            onPressed: () {
              unawaited(
                Navigator.of(context).push<void>(
                  MaterialPageRoute<void>(
                    builder: (_) =>
                        Scaffold(body: DeleteBriefingButton(onDelete: action)),
                  ),
                ),
              );
            },
            child: const Text('Open briefing'),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open briefing'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'cancel does not delete; confirmed pending deletion is single flight',
    (tester) async {
      var calls = 0;
      final pending = Completer<void>();
      await open(tester, () {
        calls++;
        return pending.future;
      });
      await tester.tap(find.text('Delete briefing'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(calls, 0);
      await tester.tap(find.text('Delete briefing'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      expect(calls, 1);
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Deleting…'))
            .onPressed,
        isNull,
      );
      pending.complete();
      await tester.pumpAndSettle();
      expect(find.text('Open briefing'), findsOneWidget);
      expect(calls, 1);
    },
  );

  testWidgets('failure retains route with safe error and explicit retry', (
    tester,
  ) async {
    var calls = 0;
    await open(tester, () async {
      if (++calls == 1) throw StateError('private filesystem path');
    });
    for (var attempt = 1; attempt <= 2; attempt++) {
      await tester.tap(find.text('Delete briefing'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      expect(calls, attempt);
      if (attempt == 1) {
        expect(
          find.text('Could not delete this briefing. Try again.'),
          findsOneWidget,
        );
        expect(find.textContaining('private filesystem'), findsNothing);
      }
    }
    expect(find.text('Open briefing'), findsOneWidget);
  });
}
