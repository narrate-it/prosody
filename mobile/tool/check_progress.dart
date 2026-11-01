import 'dart:io';
import 'package:prosody/data/models.dart';

void checkProgress() {
  final item = Briefing.fromJson({
    'id': 'item',
    'renditionId': 'rendition',
    'title': 'Briefing',
    'agent': 'Agent',
    'status': 'ready',
    'segments': [
      {
        'id': 'audio',
        'kind': 'audio',
        'durationMs': 1000,
        'url': 'https://media.example/audio',
      },
      {
        'id': 'image',
        'kind': 'image',
        'durationMs': 0,
        'images': ['https://media.example/image'],
      },
    ],
  });
  final valid = <String, dynamic>{
    'revision': 1,
    'positionSegment': 0,
    'positionMs': 500,
    'segments': [
      {'durationMs': 1000},
      {'durationMs': 0},
    ],
  };
  void rejects(Map<String, dynamic> value, {int? previous}) {
    try {
      SavedProgress.fromJson(value, item, expectedPreviousRevision: previous);
    } on FormatException {
      return;
    }
    throw StateError('Invalid progress accepted');
  }

  final saved = SavedProgress.fromJson(
    valid,
    item,
    expectedPreviousRevision: 0,
  );
  if (saved.revision != 1 || saved.segment != 0 || saved.positionMs != 500) {
    throw StateError('Saved position changed');
  }
  SavedProgress.fromJson({...valid, 'revision': 0, 'positionMs': 0}, item);
  SavedProgress.fromJson({
    ...valid,
    'positionSegment': 1,
    'positionMs': 0,
  }, item);
  for (final revision in [-1, 1.5, '1', 9007199254740992]) {
    rejects({...valid, 'revision': revision});
  }
  for (final segment in [-1, 2, '0']) {
    rejects({...valid, 'positionSegment': segment});
  }
  for (final position in [-1, 1001, 0.5, '500']) {
    rejects({...valid, 'positionMs': position});
  }
  rejects({...valid, 'positionSegment': 1, 'positionMs': 1});
  rejects({...valid, 'segments': []});
  rejects({
    ...valid,
    'segments': [
      {'durationMs': 999},
      {'durationMs': 0},
    ],
  });
  rejects({
    ...valid,
    'segments': [
      {'durationMs': 1000.0},
      {'durationMs': 0},
    ],
  });
  rejects({
    ...valid,
    'segments': [
      null,
      {'durationMs': 0},
    ],
  });
  rejects(valid, previous: 1);
  rejects({...valid, 'revision': 3}, previous: 1);
  SavedProgress.fromJson(
    {...valid, 'revision': 2},
    item,
    expectedPreviousRevision: 1,
  );
}

void main() {
  checkProgress();
  stdout.writeln(
    'Saved-progress validation passed (no service/device acceptance).',
  );
}
