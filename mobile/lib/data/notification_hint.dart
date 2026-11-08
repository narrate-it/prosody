import 'dart:convert';

/// Untrusted notification routing hints, never current access or ready-state proof.
///
/// A native tap handler must authenticate and resolve the exact publication with
/// the current service before opening anything. This type performs no IO, cache
/// lookup, navigation, session change or playback.
final class ReadyNotificationHint {
  final String feedId, itemId, renditionId;

  const ReadyNotificationHint._(this.feedId, this.itemId, this.renditionId);

  /// Admit bounded raw UTF-8 JSON before dictionary decoding can erase duplicate
  /// fields. This local profile excludes BOMs and accepts at most 4096 octets.
  /// It does not qualify a native transport or establish current access.
  factory ReadyNotificationHint.fromJsonBytes(List<int> bytes) {
    const invalid = FormatException('Invalid ready notification');
    if (bytes.isEmpty ||
        bytes.length > 4096 ||
        bytes.any((byte) => byte < 0 || byte > 255)) {
      throw invalid;
    }
    if (bytes.length >= 3 &&
        bytes[0] == 0xef &&
        bytes[1] == 0xbb &&
        bytes[2] == 0xbf) {
      throw invalid;
    }
    try {
      final value = _ReadyJson(utf8.decode(bytes, allowMalformed: false))
          .read();
      return ReadyNotificationHint.fromPayload(value);
    } on FormatException {
      // Decoder exceptions can retain private input and offsets. Never expose
      // those diagnostics through the public hint boundary.
      throw invalid;
    }
  }

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

/// Closed-profile JSON grammar: root objects and their two declared child maps
/// only. There is no arbitrary graph recursion; unknown keys fail before values.
final class _ReadyJson {
  final String source;
  int offset = 0;
  _ReadyJson(this.source);

  /// Require one complete envelope, permitting only ordinary JSON whitespace.
  Map<String, Object?> read() {
    final value = _object('root');
    _space();
    if (offset != source.length) {
      _invalid();
    }
    return value;
  }

  /// Track decoded names so raw and escaped spellings cannot hide duplicates.
  Map<String, Object?> _object(String kind) {
    final keys = switch (kind) {
      'root' => const {'aps', 'prosody'},
      'aps' => const {'alert'},
      _ => const {'version', 'kind', 'feedId', 'itemId', 'renditionId'},
    };
    _expect(0x7b);
    final value = <String, Object?>{};
    if (_take(0x7d)) {
      return value;
    }
    while (true) {
      final key = _string();
      if (!keys.contains(key) || value.containsKey(key)) {
        _invalid();
      }
      _expect(0x3a);
      if (kind == 'root') {
        value[key] = _object(key);
      } else if (key == 'version') {
        // JSON 1.0/1e0 decode as doubles, not the selected integer version.
        _expect(0x31);
        value[key] = 1;
      } else {
        value[key] = _string();
      }
      if (_take(0x7d)) {
        return value;
      }
      _expect(0x2c);
    }
  }

  /// Decode exactly one JSON string, requiring paired Unicode surrogates.
  String _string() {
    _space();
    final start = offset;
    _expect(0x22);
    while (offset < source.length) {
      final unit = source.codeUnitAt(offset++);
      if (unit == 0x22) {
        final value = jsonDecode(source.substring(start, offset)) as String;
        for (var i = 0; i < value.length; i++) {
          final code = value.codeUnitAt(i);
          if (code >= 0xd800 && code <= 0xdbff) {
            if (++i == value.length) {
              _invalid();
            }
            final low = value.codeUnitAt(i);
            if (low < 0xdc00 || low > 0xdfff) {
              _invalid();
            }
          } else if (code >= 0xdc00 && code <= 0xdfff) {
            _invalid();
          }
        }
        return value;
      }
      if (unit < 0x20) {
        _invalid();
      }
      if (unit == 0x5c) {
        if (offset == source.length) {
          _invalid();
        }
        offset++;
      }
    }
    _invalid();
  }

  /// Consume only space, tab, CR and LF, never other Unicode separators.
  void _space() {
    while (offset < source.length &&
        const {0x20, 0x09, 0x0a, 0x0d}.contains(source.codeUnitAt(offset))) {
      offset++;
    }
  }

  /// Consume a structural character after bounded whitespace if present.
  bool _take(int unit) {
    _space();
    if (offset == source.length || source.codeUnitAt(offset) != unit) {
      return false;
    }
    offset++;
    return true;
  }

  /// Require the next structural character without accepting a prefix token.
  void _expect(int unit) {
    if (!_take(unit)) {
      _invalid();
    }
  }

  /// Keep parser failures fixed; the public boundary also normalizes JSON errors.
  Never _invalid() => throw const FormatException('Invalid ready notification');
}
