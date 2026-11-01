import '../data/models.dart';

// Append only new playable items in server order. Keep an already queued item's
// rendition stable so a later page cannot replace content during playback.
List<Briefing> newPlayableItems(
  Iterable<Briefing> existing,
  Iterable<Briefing> incoming,
) {
  final ids = existing.map((item) => item.id).toSet();
  return incoming
      .where(
        (item) =>
            item.status == 'ready' &&
            item.segments.isNotEmpty &&
            ids.add(item.id),
      )
      .toList(growable: false);
}
