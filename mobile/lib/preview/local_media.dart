import 'dart:io';

// Only paths registered by this preview's generator can be removed. Unknown
// IDs and bundled samples never become filesystem paths from caller input.
Future<void> deletePreviewMedia(Map<String, String> files, String id) async {
  final url = 'https://preview.invalid/$id.mp3';
  final path = files[url];
  if (path == null) return;
  try {
    await File(path).delete();
  } on FileSystemException catch (failure) {
    // iOS/macOS ENOENT: an already absent owned file is an idempotent delete.
    if (failure.osError?.errorCode != 2) rethrow;
  }
  files.remove(url);
}
