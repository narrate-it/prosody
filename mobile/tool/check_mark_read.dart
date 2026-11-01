import 'dart:io';
import 'package:prosody/data/models.dart';
import 'package:prosody/data/mark_read.dart';

void checkMarkRead() {
  final item = Briefing.fromJson({
    'id': 'item',
    'renditionId': 'rendition',
    'title': 'Title',
    'agent': 'Agent',
    'status': 'ready',
    'segments': [
      {
        'id': 'audio',
        'kind': 'audio',
        'durationMs': 1000,
        'url': 'https://media.example/audio',
      },
    ],
  });
  final saved = SavedProgress.fromJson({
    'revision': 8,
    'positionSegment': 0,
    'positionMs': 400,
    'segments': [
      {'durationMs': 1000},
    ],
  }, item);
  final update = markReadUpdate(item, saved);
  if (update['renditionId'] != 'rendition' ||
      update['expectedRevision'] != 8 ||
      update['segment'] != 0 ||
      update['positionMs'] != 400 ||
      update['markedRead'] != true ||
      update['viewed'] != false ||
      (update['played'] as List).isNotEmpty) {
    throw StateError('Mark-read changed position or invented consumption');
  }
  if (update.containsKey('personId') || update.containsKey('workspaceId')) {
    throw StateError('Client attempted to supply authority');
  }
}

void main() {
  checkMarkRead();
  stdout.writeln('Mark-read payload checks passed; HTTP/UI behavior untested.');
}
