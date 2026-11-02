import 'dart:async';
import 'package:http/http.dart' as http;
import 'api.dart';
import 'models.dart';

// Resume position is optional; current authority is not. A failed or expired
// access check must still prevent selecting a previously fetched feed page.
Future<SavedProgress?> optionalResume(ProsodyApi api, Briefing item) async {
  try {
    return SavedProgress.fromJson(
      await api.progress(item.id, item.renditionId),
      item,
    );
  } on ApiFailure catch (error) {
    if (error.status == 0 ||
        error.status == 404 ||
        error.status == 408 ||
        error.status == 429 ||
        error.status >= 500) {
      return null;
    }
    rethrow;
  } on FormatException {
    return null;
  } on TimeoutException {
    return null;
  } on http.ClientException {
    return null;
  }
}
