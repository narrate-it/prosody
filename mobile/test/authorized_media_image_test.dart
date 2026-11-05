import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/playback/authorized_media_buffer.dart';
import 'package:prosody/playback/authorized_media_image.dart';

/// Enforces the shared-budget boundary while gates control only this widget's
/// preparation. Files are prewritten so the gate does not depend on real time.
class ImageBudgetBuffer extends Fake implements AuthorizedMediaBuffer {
  final Map<Uri, File> files;
  final Uri gatedUri;
  final gate = Completer<void>();
  final started = Completer<void>();
  final events = StreamController<void>.broadcast();
  final requests = <Uri>[];
  final reservations = <String>{};
  final released = <String>[];
  final prepared = <Uri>{};
  final timeline = <String>[];
  int cancellations = 0;
  String? failReleaseOncePath;
  ImageBudgetBuffer(this.files, this.gatedUri);
  @override
  bool get isClosed => false;
  @override
  Stream<void> get accessEnded => events.stream;
  @override
  void cancelPending() {
    cancellations++;
  }

  @override
  Future<File> prepare(Uri uri) async {
    requests.add(uri);
    timeline.add('prepare ${uri.path}');
    if (reservations.length >= 2) {
      timeline.add('budget rejected ${uri.path}');
      throw StateError('shared buffer full');
    }
    final file = files[uri]!;
    reservations.add(file.path);
    if (uri == gatedUri) {
      started.complete();
      await gate.future;
    }
    prepared.add(uri);
    timeline.add('prepared ${uri.path}');
    return file;
  }

  @override
  Future<void> release(File file) async {
    timeline.add('release ${file.path}');
    if (file.path == failReleaseOncePath) {
      failReleaseOncePath = null;
      timeline.add('release failed ${file.path}');
      throw StateError('injected image release failure');
    }
    reservations.remove(file.path);
    released.add(file.path);
    timeline.add('released ${file.path}');
  }
}

/// Observe completion rather than treating two frames as an async cleanup
/// contract. FileImage uses async provider/cache work and real filesystem reads;
/// pump drains fake-zone work, while runAsync lets real-zone completions arrive.
Future<void> pumpUntilPrepared(
  WidgetTester tester,
  ImageBudgetBuffer buffer,
  Uri uri,
) async {
  for (var turn = 0; turn < 30; turn++) {
    await tester.pump();
    if (buffer.prepared.contains(uri)) {
      await tester.pump();
      return;
    }
    await tester.runAsync(() async {
      await Future<void>.delayed(const Duration(milliseconds: 1));
    });
  }
  fail(
    'Replacement did not complete bounded async cleanup. Timeline: ${buffer.timeline}',
  );
}

void main() {
  for (final failRelease in [false, true]) {
    testWidgets(
      'image replacement joins stale preparation and releases budget with transient release failure=$failRelease without shared cancellation',
      (tester) async {
        final oldUri = Uri.parse('https://media.example/old');
        final newUri = Uri.parse('https://media.example/new');
        final unrelatedUri = Uri.parse('https://media.example/unrelated');
        late Directory directory;
        final files = <Uri, File>{};
        await tester.runAsync(() async {
          directory = await Directory.systemTemp.createTemp(
            'prosody-image-test-',
          );
          final png = base64Decode(
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jfZkAAAAASUVORK5CYII=',
          );
          var serial = 0;
          for (final uri in [oldUri, newUri, unrelatedUri]) {
            files[uri] = await File('${directory.path}/${serial++}.png')
                .writeAsBytes(png);
          }
        });
        final buffer = ImageBudgetBuffer(files, oldUri);
        final unrelated = await buffer.prepare(unrelatedUri);
        Widget image(Uri uri) => MaterialApp(
          home: AuthorizedMediaImage(
            key: const ValueKey('stable-image'),
            buffer: buffer,
            uri: uri,
          ),
        );
        await tester.pumpWidget(image(oldUri));
        await tester.pump();
        expect(buffer.started.isCompleted, isTrue);
        await tester.pumpWidget(image(newUri));
        await tester.pump();
        expect(
          tester
              .widget<AuthorizedMediaImage>(find.byType(AuthorizedMediaImage))
              .uri,
          newUri,
          reason: 'The current widget must receive the replacement before opening the old gate',
        );
        expect(buffer.requests, [unrelatedUri, oldUri]);
        expect(buffer.reservations.length, 2);
        if (failRelease) {
          buffer.failReleaseOncePath = files[oldUri]!.path;
        }
        buffer.gate.complete();
        await pumpUntilPrepared(tester, buffer, newUri);
        expect(buffer.requests, [unrelatedUri, oldUri, newUri]);
        expect(buffer.released, contains(files[oldUri]!.path));
        expect(
          buffer.timeline.indexOf('released ${files[oldUri]!.path}'),
          lessThan(buffer.timeline.indexOf('prepare ${newUri.path}')),
          reason: 'Release must actually finish before the replacement request',
        );
        expect(
          buffer.timeline.where((event) => event.startsWith('budget rejected')),
          isEmpty,
        );
        expect(
          buffer.timeline
              .where((event) => event.startsWith('release failed'))
              .length,
          failRelease ? 1 : 0,
        );
        expect(
          buffer.reservations,
          containsAll([unrelated.path, files[newUri]!.path]),
        );
        expect(buffer.cancellations, 0);
        expect(find.byType(Image), findsOneWidget);
        await tester.pumpWidget(const SizedBox());
        await tester.pump();
        await buffer.release(unrelated);
        await buffer.events.close();
        await tester.runAsync(() async {
          await directory.delete(recursive: true);
        });
      },
    );
  }
}
