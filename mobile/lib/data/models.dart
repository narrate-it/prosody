import 'dart:convert';

class Feed {
  final String id, name, kind;
  final bool canManage;
  const Feed({
    required this.id,
    required this.name,
    required this.kind,
    required this.canManage,
  });
  factory Feed.fromJson(Map<String, dynamic> j) => Feed(
    id: j['id'] as String,
    name: j['name'] as String,
    kind: j['kind'] as String,
    canManage: j['canManage'] as bool,
  );
}

class Segment {
  final String id, kind;
  final Uri? media;
  final List<Uri> images;
  final Duration duration;
  const Segment({
    required this.id,
    required this.kind,
    this.media,
    this.images = const [],
    required this.duration,
  });
  factory Segment.fromJson(Map<String, dynamic> j) {
    final id = j['id'], kind = j['kind'], milliseconds = j['durationMs'];
    if (id is! String ||
        id.isEmpty ||
        !const ['narration', 'audio', 'video', 'image'].contains(kind) ||
        milliseconds is! int ||
        milliseconds < 0 ||
        (kind != 'image' && milliseconds == 0)) {
      throw const FormatException('Invalid playback segment');
    }
    final duration = Duration(milliseconds: milliseconds);
    if (duration.inMilliseconds != milliseconds) {
      throw const FormatException('Unrepresentable playback duration');
    }
    final media = j['url'] == null ? null : privateMediaUri(j['url'] as String);
    final images = (j['images'] as List? ?? [])
        .map((e) => privateMediaUri(e as String))
        .toList();
    if ((kind == 'image' &&
            (images.isEmpty || media != null || milliseconds != 0)) ||
        (kind != 'image' && media == null)) {
      throw const FormatException('Missing or inconsistent segment media');
    }
    return Segment(
      id: id,
      kind: kind as String,
      media: media,
      images: List.unmodifiable(images),
      duration: duration,
    );
  }
}

Uri privateMediaUri(String value) {
  final uri = Uri.parse(value);
  if (uri.scheme != 'https' ||
      uri.host.isEmpty ||
      uri.userInfo.isNotEmpty ||
      uri.hasFragment) {
    throw const FormatException('Invalid media URL');
  }
  return uri;
}

class Briefing {
  final String id, renditionId, title, agent, status, source, script;
  final List<Segment> segments;
  const Briefing({
    required this.id,
    required this.renditionId,
    required this.title,
    required this.agent,
    required this.status,
    required this.source,
    required this.script,
    required this.segments,
  });
  factory Briefing.fromJson(Map<String, dynamic> j) {
    final id = j['id'], status = j['status'];
    if (id is! String ||
        id.isEmpty ||
        !const ['processing', 'failed', 'ready'].contains(status) ||
        j['title'] is! String ||
        j['agent'] is! String ||
        (j['source'] != null && j['source'] is! String) ||
        (j['script'] != null && j['script'] is! String) ||
        (j['renditionId'] != null && j['renditionId'] is! String) ||
        (j['segments'] != null && j['segments'] is! List)) {
      throw const FormatException('Invalid briefing');
    }
    final segments = (j['segments'] as List? ?? []).map((e) {
      if (e is! Map<String, dynamic>) {
        throw const FormatException('Invalid playback segment');
      }
      return Segment.fromJson(e);
    }).toList();
    final renditionId = j['renditionId'] as String? ?? '';
    final ids = <String>{};
    var generatedMs = 0;
    var totalDuration = Duration.zero;
    for (final segment in segments) {
      if (!ids.add(segment.id)) {
        throw const FormatException('Duplicate playback segment');
      }
      if (segment.kind == 'narration') {
        generatedMs += segment.duration.inMilliseconds;
        if (generatedMs > 600000) {
          throw const FormatException('Invalid playable rendition');
        }
      }
      final nextDuration = totalDuration + segment.duration;
      if (nextDuration < totalDuration) {
        throw const FormatException('Unrepresentable playback timeline');
      }
      totalDuration = nextDuration;
    }
    if (j['status'] == 'ready' && (renditionId.isEmpty || segments.isEmpty)) {
      throw const FormatException('Invalid playable rendition');
    }
    return Briefing(
      id: id,
      renditionId: renditionId,
      title: j['title'] as String,
      agent: j['agent'] as String,
      status: status as String,
      source: j['source'] as String? ?? '',
      script: j['script'] as String? ?? '',
      segments: List.unmodifiable(segments),
    );
  }
  Duration get duration =>
      segments.fold(Duration.zero, (a, b) => a + b.duration);
}

class FeedPage {
  final List<Briefing> items;
  final String? nextCursor;
  const FeedPage(this.items, this.nextCursor);
  factory FeedPage.fromJson(Map<String, dynamic> j) {
    final items = j['items'], cursor = j['nextCursor'];
    if (items is! List ||
        (cursor != null && (cursor is! String || cursor.isEmpty))) {
      throw const FormatException('Invalid feed page');
    }
    final briefings = <Briefing>[];
    for (final item in items) {
      if (item is! Map<String, dynamic>) {
        throw const FormatException('Invalid feed entry');
      }
      briefings.add(Briefing.fromJson(item));
    }
    return FeedPage(List.unmodifiable(briefings), cursor as String?);
  }
}

class ProviderModel {
  final String id, name;
  final List<String> voices;
  const ProviderModel(this.id, this.name, this.voices);
  factory ProviderModel.fromJson(Map<String, dynamic> j) {
    final id = j['id'], name = j['name'];
    final rawVoices = j['voices'] ?? const [];
    if (id is! String ||
        id.trim().isEmpty ||
        name is! String ||
        name.trim().isEmpty ||
        rawVoices is! List) {
      throw const FormatException('Invalid provider model');
    }
    final voices = <String>[];
    final seen = <String>{};
    for (final voice in rawVoices) {
      if (voice is! String || voice.trim().isEmpty || !seen.add(voice)) {
        throw const FormatException('Invalid or duplicate provider voice');
      }
      voices.add(voice);
    }
    return ProviderModel(id, name, List.unmodifiable(voices));
  }
}

List<ProviderModel> providerCatalogFromJson(Object? value) {
  if (value is! List) {
    throw const FormatException('Invalid provider catalog');
  }
  final ids = <String>{};
  final models = <ProviderModel>[];
  for (final entry in value) {
    if (entry is! Map<String, dynamic>) {
      throw const FormatException('Invalid provider catalog entry');
    }
    final model = ProviderModel.fromJson(entry);
    if (!ids.add(model.id)) {
      throw const FormatException('Duplicate provider model');
    }
    models.add(model);
  }
  return List.unmodifiable(models);
}

class FeedSettings {
  final int revision;
  final String? textModel, speechModel, voice;
  final bool hasKey, notifications;
  const FeedSettings({
    required this.revision,
    this.textModel,
    this.speechModel,
    this.voice,
    required this.hasKey,
    required this.notifications,
  });
  factory FeedSettings.fromJson(Map<String, dynamic> j) {
    final revision = j['revision'];
    final hasKey = j['hasKey'], notifications = j['notifications'];
    if (revision is! int ||
        revision < 0 ||
        revision > 9007199254740991 ||
        hasKey is! bool ||
        notifications is! bool) {
      throw const FormatException('Invalid feed settings');
    }
    String? selection(String key) {
      final value = j[key];
      if (value == null) {
        return null;
      }
      if (value is! String || value.trim().isEmpty) {
        throw const FormatException('Invalid provider selection');
      }
      return value;
    }

    return FeedSettings(
      revision: revision,
      textModel: selection('textModel'),
      speechModel: selection('speechModel'),
      voice: selection('voice'),
      hasKey: hasKey,
      notifications: notifications,
    );
  }
}

Map<String, dynamic> objectFromJson(String body) {
  final value = jsonDecode(body);
  if (value is! Map<String, dynamic>) {
    throw const FormatException('Expected an object');
  }
  return value;
}

// Validate the response against the requested immutable rendition. This is not
// an authority check; the service must bind the authenticated person and item.
class SavedProgress {
  final int revision, segment, positionMs;
  const SavedProgress(this.revision, this.segment, this.positionMs);

  factory SavedProgress.fromJson(
    Map<String, dynamic> j,
    Briefing item, {
    int? expectedPreviousRevision,
  }) {
    final revision = j['revision'];
    final segment = j['positionSegment'];
    final position = j['positionMs'];
    final segments = j['segments'];
    // Preserve exact revision integers on both native Dart and Flutter web.
    if (revision is! int ||
        revision < 0 ||
        revision > 9007199254740991 ||
        (expectedPreviousRevision != null &&
            revision != expectedPreviousRevision + 1) ||
        segment is! int ||
        segment < 0 ||
        segment >= item.segments.length ||
        position is! int ||
        position < 0 ||
        position > item.segments[segment].duration.inMilliseconds ||
        segments is! List ||
        segments.length != item.segments.length) {
      throw const FormatException('Invalid saved progress');
    }
    for (var i = 0; i < segments.length; i++) {
      final saved = segments[i];
      if (saved is! Map<String, dynamic> ||
          saved['durationMs'] is! int ||
          saved['durationMs'] != item.segments[i].duration.inMilliseconds) {
        throw const FormatException('Progress does not match rendition');
      }
    }
    return SavedProgress(revision, segment, position);
  }
}
