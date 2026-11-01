import 'dart:io';
import 'package:prosody/data/models.dart';
import 'package:prosody/playback/queue.dart';

void checkQueue() {
  Briefing item(
    String id, {
    String status = 'ready',
    String rendition = 'v1',
    bool playable = true,
  }) => Briefing(
    id: id,
    renditionId: rendition,
    title: id,
    agent: 'agent',
    status: status,
    source: '',
    script: '',
    segments: playable
        ? [
            Segment(
              id: 'segment',
              kind: 'audio',
              media: Uri.parse('https://media.example/a'),
              duration: const Duration(seconds: 1),
            ),
          ]
        : [],
  );
  void expectIds(List<Briefing> actual, List<String> expected) {
    if (actual.map((e) => e.id).toList().toString() != expected.toString()) {
      throw StateError(
        'Queue IDs differ: ${actual.map((e) => e.id)} vs $expected',
      );
    }
  }

  final original = item('a');
  final existing = [original];
  final incoming = [
    item('a', rendition: 'v2'),
    item('b'),
    item('b'),
    item('c', status: 'processing', playable: false),
    item('d'),
  ];
  final added = newPlayableItems(existing, incoming);
  expectIds(added, ['b', 'd']);
  expectIds(existing, ['a']);
  if (!identical(existing.first, original) || incoming.length != 5) {
    throw StateError('Input queue mutated');
  }
  final queue = [...existing, ...added];
  expectIds(newPlayableItems(queue, incoming), []);
  expectIds(newPlayableItems(queue, [item('c')]), ['c']);
  expectIds(newPlayableItems([], [item('a'), item('a'), item('b')]), [
    'a',
    'b',
  ]);
  expectIds(
    newPlayableItems([], [
      item('a', status: 'failed'),
      item('a', playable: false),
      item('a'),
    ]),
    ['a'],
  );
  final first = item('new', rendition: 'first');
  final result = newPlayableItems([], [first, item('new', rendition: 'later')]);
  if (result.length != 1 || !identical(result.single, first)) {
    throw StateError('First queued rendition replaced');
  }
  expectIds(newPlayableItems(queue, []), []);
}

void main() {
  checkQueue();
  stdout.writeln(
    'Queue pagination checks passed (no platform playback exercised).',
  );
}
