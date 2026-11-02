import 'dart:async';
import 'package:http/http.dart' as http;
import 'api.dart';
import 'models.dart';

// Resume position is optional; current authority is not. A failed or expired
// access check must still prevent selecting a previously fetched feed page.
class ResumeLookup {
  final SavedProgress? saved;
  final bool unknown;
  const ResumeLookup.saved(SavedProgress progress)
    : saved = progress,
      unknown = false;
  const ResumeLookup.absent() : saved = null, unknown = false;
  const ResumeLookup.unknown() : saved = null, unknown = true;
}

Future<ResumeLookup> optionalResume(ProsodyApi api, Briefing item) async {
  try {
    return ResumeLookup.saved(
      SavedProgress.fromJson(
        await api.progress(item.id, item.renditionId),
        item,
      ),
    );
  } on ApiFailure catch (error) {
    if (error.status == 404) return const ResumeLookup.absent();
    if (error.status == 0 ||
        error.status == 408 ||
        error.status == 429 ||
        error.status >= 500) {
      return const ResumeLookup.unknown();
    }
    rethrow;
  } on FormatException {
    return const ResumeLookup.unknown();
  } on TimeoutException {
    return const ResumeLookup.unknown();
  } on http.ClientException {
    return const ResumeLookup.unknown();
  }
}
