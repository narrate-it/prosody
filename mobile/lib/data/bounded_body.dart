import 'dart:async';

class BodyTooLarge implements Exception {
  const BodyTooLarge();
}

// One absolute body deadline, not a deadline reset by every arriving chunk.
// Completion cancels collection; transport-specific socket abort remains owned
// by the HTTP client. No body bytes appear in error messages.
Future<List<int>> collectBoundedBody(
  Stream<List<int>> stream, {
  required int maxBytes,
  required Duration deadline,
}) {
  if (maxBytes <= 0 || deadline <= Duration.zero) {
    throw ArgumentError('Positive body limits required');
  }
  final result = Completer<List<int>>();
  final bytes = <int>[];
  StreamSubscription<List<int>>? subscription;
  Timer? timer;
  void cancel() {
    timer?.cancel();
    final current = subscription;
    subscription = null;
    if (current != null) {
      unawaited(current.cancel().catchError((Object _) {}));
    }
  }

  void fail(Object error, [StackTrace? stack]) {
    if (result.isCompleted) {
      return;
    }
    result.completeError(error, stack);
    cancel();
  }

  timer = Timer(
    deadline,
    () => fail(TimeoutException('Response body deadline exceeded')),
  );
  try {
    subscription = stream.listen(
      (chunk) {
        if (result.isCompleted) {
          return;
        }
        if (chunk.length > maxBytes - bytes.length) {
          fail(const BodyTooLarge());
          return;
        }
        bytes.addAll(chunk);
      },
      onError: (Object error, StackTrace stack) => fail(error, stack),
      onDone: () {
        if (!result.isCompleted) {
          result.complete(bytes);
        }
        cancel();
      },
    );
  } catch (error, stack) {
    fail(error, stack);
  }
  // Also handle streams that finish synchronously while listen is returning.
  if (result.isCompleted) {
    cancel();
  }
  return result.future;
}
