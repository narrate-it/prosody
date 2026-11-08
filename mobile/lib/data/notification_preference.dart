import 'api.dart';
import 'models.dart';

typedef NotificationPreferenceResult = ({
  FeedSettings settings,
  bool writeConfirmed,
});

/// Reconciles one local preference action without repeating its write.
/// Callbacks carry transport scope, not native authority or commit proof.
Future<NotificationPreferenceResult> confirmNotificationPreference({
  required Future<void> Function() write,
  required Future<FeedSettings> Function() read,
  required Future<void> Function() checkSession,
  bool Function()? isCurrent,
}) async {
  void checkAction() {
    if (isCurrent != null && !isCurrent()) {
      throw const ApiFailure(0, 'This preference action is no longer current.');
    }
  }

  checkAction();
  Object? writeError;
  try {
    await write();
  } catch (error) {
    writeError = error;
  }
  await checkSession();
  checkAction();
  if (writeError is ApiFailure &&
      (writeError.status == 401 || writeError.status == 403)) {
    throw writeError;
  }
  final settings = await read();
  await checkSession();
  checkAction();
  return (settings: settings, writeConfirmed: writeError == null);
}
