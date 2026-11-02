import 'dart:async';
import 'dart:io';
import 'package:prosody/data/bounded_body.dart';

Future<void> checkBoundedBody() async {
  final bytes = await collectBoundedBody(
    Stream.fromIterable([
      [1, 2],
      [3],
    ]),
    maxBytes: 3,
    deadline: const Duration(seconds: 1),
  );
  if (bytes.join() != '123') throw StateError('Changed response bytes');
  var canceled = false;
  final large = StreamController<List<int>>(
    onCancel: () {
      canceled = true;
    },
  );
  final oversized = collectBoundedBody(
    large.stream,
    maxBytes: 2,
    deadline: const Duration(seconds: 1),
  );
  large.add([1, 2, 3]);
  try {
    await oversized;
    throw StateError('Oversized body accepted');
  } on BodyTooLarge {
    /* expected */
  }
  if (!canceled) throw StateError('Oversized collection not canceled');
  await large.close();

  var delivered = 0;
  var stopped = false;
  Timer? ticker;
  final trickle = StreamController<List<int>>(
    onCancel: () {
      stopped = true;
      ticker?.cancel();
    },
  );
  ticker = Timer.periodic(const Duration(milliseconds: 5), (_) {
    delivered++;
    trickle.add([1]);
  });
  try {
    await collectBoundedBody(
      trickle.stream,
      maxBytes: 10000,
      deadline: const Duration(milliseconds: 40),
    );
    throw StateError('Trickling response never reached deadline');
  } on TimeoutException {
    /* expected */
  } finally {
    ticker.cancel();
    await trickle.close();
  }
  if (!stopped || delivered == 0) {
    throw StateError('Deadline did not stop trickling body');
  }

  final empty = await collectBoundedBody(
    const Stream.empty(),
    maxBytes: 1,
    deadline: const Duration(seconds: 1),
  );
  if (empty.isNotEmpty) throw StateError('Empty response changed');
  final failure = StateError('test transport');
  try {
    await collectBoundedBody(
      Stream.error(failure),
      maxBytes: 1,
      deadline: const Duration(seconds: 1),
    );
    throw StateError('Transport error lost');
  } catch (error) {
    if (!identical(error, failure)) rethrow;
  }
}

Future<void> main() async {
  await checkBoundedBody();
  stdout.writeln('Bounded body checks passed; no HTTP service exercised.');
}
