import 'dart:io';
// Dependency-free response validation; no server or media decoder is exercised.
import 'package:prosody/data/models.dart';

void checkModels() {
  Map<String, dynamic> segment(String id, String kind, int duration) => {
    'id': id,
    'kind': kind,
    'durationMs': duration,
    if (kind != 'image') 'url': 'https://media.example/asset?signature=opaque',
    if (kind == 'image') 'images': ['https://media.example/image'],
  };
  Map<String, dynamic> item(List<Map<String, dynamic>> segments) => {
    'id': 'item',
    'renditionId': 'revision',
    'title': 'Briefing',
    'agent': 'Agent',
    'status': 'ready',
    'segments': segments,
  };
  void rejects(void Function() action) {
    try {
      action();
    } on FormatException {
      return;
    }
    throw StateError('Malformed playback input accepted');
  }

  Briefing.fromJson(item([segment('one', 'narration', 600000)]));
  // Each supplied clip fits Duration independently, but their combined native
  // timeline does not. Supplied media still has no ten-minute product limit.
  const largeClipMs = 5000000000000000;
  final longItem = Briefing.fromJson(
    item([segment('one', 'audio', largeClipMs)]),
  );
  if (longItem.duration.inMilliseconds != largeClipMs) {
    throw StateError('Representable supplied duration changed');
  }
  rejects(
    () => Briefing.fromJson(
      item([
        segment('one', 'audio', largeClipMs),
        segment('two', 'video', largeClipMs),
      ]),
    ),
  );
  Briefing.fromJson(
    item([
      segment('one', 'audio', 7200000),
      segment('two', 'video', 7200000),
      segment('three', 'image', 0),
    ]),
  );
  Briefing.fromJson({...item([]), 'status': 'processing', 'renditionId': null});
  for (final invalid in <Map<String, dynamic>>[
    {'id': ''},
    {'id': null},
    {'status': 'unknown'},
    {'status': null},
    {'title': 42},
    {'agent': null},
    {'source': false},
    {'script': []},
    {'renditionId': 1},
    {'segments': 'invalid'},
    {
      'segments': [null],
    },
  ]) {
    rejects(
      () => Briefing.fromJson({
        ...item([segment('one', 'audio', 10)]),
        ...invalid,
      }),
    );
  }
  for (final invalid in <Map<String, dynamic>>[
    {},
    {
      'items': [null],
    },
    {'items': [], 'nextCursor': ''},
    {'items': [], 'nextCursor': 1},
  ]) {
    rejects(() => FeedPage.fromJson(invalid));
  }
  final page = FeedPage.fromJson({
    'items': [
      item([segment('one', 'audio', 10)]),
    ],
    'nextCursor': 'opaque cursor',
  });
  if (page.items.single.id != 'item' || page.nextCursor != 'opaque cursor') {
    throw StateError('Valid feed page changed');
  }
  var immutable = false;
  try {
    page.items.clear();
  } on UnsupportedError {
    immutable = true;
  }
  if (!immutable) throw StateError('Decoded page is mutable');
  rejects(() => Briefing.fromJson(item([])));
  rejects(
    () => Briefing.fromJson({
      ...item([segment('one', 'audio', 10)]),
      'renditionId': '',
    }),
  );
  rejects(
    () => Briefing.fromJson(
      item([
        segment('one', 'narration', 300001),
        segment('two', 'narration', 300000),
      ]),
    ),
  );
  rejects(
    () => Briefing.fromJson(
      item([segment('one', 'audio', 10), segment('one', 'audio', 20)]),
    ),
  );
  for (final kind in ['audio', 'video', 'narration']) {
    rejects(() => Segment.fromJson(segment('one', kind, -1)));
    rejects(() => Segment.fromJson(segment('one', kind, 0)));
    rejects(
      () => Segment.fromJson({...segment('one', kind, 100), 'url': null}),
    );
  }
  rejects(() => Segment.fromJson(segment('one', 'unknown', 100)));
  rejects(() => Segment.fromJson(segment('', 'audio', 100)));
  rejects(() => Segment.fromJson(segment('one', 'image', 100)));
  rejects(
    () => Segment.fromJson({...segment('one', 'image', 0), 'images': []}),
  );
  for (final url in [
    'http://media.example/a',
    'https://user:pass@media.example/a',
    'https://media.example/a#fragment',
    '/relative',
  ]) {
    rejects(
      () => Segment.fromJson({...segment('one', 'audio', 100), 'url': url}),
    );
  }
}

void main() {
  checkModels();
  stdout.writeln(
    'Playback response validation passed (no service/device acceptance).',
  );
}
