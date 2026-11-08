/// Untrusted notification routing hints, never current access or ready-state proof.
///
/// A native tap handler must authenticate and resolve the exact publication with
/// the current service before opening anything. This type performs no IO, cache
/// lookup, navigation, session change or playback.
final class ReadyNotificationHint {
  final String feedId, itemId, renditionId;

  const ReadyNotificationHint._(this.feedId, this.itemId, this.renditionId);

  /// Decode only the closed version-one payload emitted by Prosody's Go codec.
  /// Input is the decoded payload dictionary, not a raw JSON string or URL.
  factory ReadyNotificationHint.fromPayload(Object? payload) {
    if (payload is! Map<Object?, Object?> ||
        !_exactKeys(payload, const {'aps', 'prosody'})) {
      throw const FormatException('Invalid ready notification');
    }
    final aps = payload['aps'], routing = payload['prosody'];
    if (aps is! Map<Object?, Object?> ||
        !_exactKeys(aps, const {'alert'}) ||
        aps['alert'] is! String ||
        aps['alert'] != 'A new briefing is ready.' ||
        routing is! Map<Object?, Object?> ||
        !_exactKeys(routing, const {
          'version',
          'kind',
          'feedId',
          'itemId',
          'renditionId',
        })) {
      throw const FormatException('Invalid ready notification');
    }
    final version = routing['version'], kind = routing['kind'];
    final feed = routing['feedId'], item = routing['itemId'];
    final rendition = routing['renditionId'];
    if (version is! int ||
        version != 1 ||
        kind is! String ||
        kind != 'ready' ||
        feed is! String ||
        item is! String ||
        rendition is! String ||
        !_canonicalUUID(feed) ||
        feed[14] != '7' ||
        !const {'8', '9', 'a', 'b'}.contains(feed[19]) ||
        !_canonicalUUID(item) ||
        !_canonicalUUID(rendition)) {
      throw const FormatException('Invalid ready notification');
    }
    return ReadyNotificationHint._(feed, item, rendition);
  }
}

/// Refuse extensions or non-string platform keys without recursively walking data.
bool _exactKeys(Map<Object?, Object?> value, Set<String> expected) =>
    value.length == expected.length &&
    value.keys.every((key) => key is String && expected.contains(key));

final _uuidShape = RegExp(
  r'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$',
);

/// Matches the Go codec's nonzero canonical shape, not database identity.
bool _canonicalUUID(String value) =>
    value.length == 36 &&
    value != '00000000-0000-0000-0000-000000000000' &&
    _uuidShape.hasMatch(value);
