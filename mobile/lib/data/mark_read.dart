import 'models.dart';

// Explicit human action, independent of listened coverage. The service still
// binds the person/item/rendition and enforces the revision transactionally.
Map<String, dynamic> markReadUpdate(Briefing item, SavedProgress progress) => {
  'renditionId': item.renditionId,
  'expectedRevision': progress.revision,
  'segment': progress.segment,
  'positionMs': progress.positionMs,
  'played': <Map<String, int>>[],
  'viewed': false,
  'markedRead': true,
};
