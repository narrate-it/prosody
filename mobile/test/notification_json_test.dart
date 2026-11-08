import 'dart:collection';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/data/notification_hint.dart';

final _golden = File('../testdata/notifications/ready-v1.json')
    .readAsBytesSync();
final _json = utf8.decode(_golden).trim();

// Oversized inputs must be refused by length before any byte access or decoding.
class _Oversized extends ListBase<int> {
  @override
  int get length => 4097;
  @override
  set length(int value) => throw UnsupportedError('Immutable test input');
  @override
  int operator [](int index) => throw StateError('Oversized bytes were read');
  @override
  void operator []=(int index, int value) =>
      throw UnsupportedError('Immutable test input');
}

void main() {
  test('shared Go JSON bytes produce the same immutable hint', () {
    final bytes = List<int>.from(_golden);
    final hint = ReadyNotificationHint.fromJsonBytes(bytes);
    bytes.fillRange(0, bytes.length, 0);
    expect(hint.feedId, '019a3f00-1234-7000-8000-000000000001');
    expect(hint.itemId, '019a3f00-1234-7000-8000-000000000002');
    expect(hint.renditionId, '019a3f00-1234-7000-8000-000000000003');
  });
  test('escaped known field names preserve the selected profile', () {
    final value = _json
        .replaceFirst('"aps"', r'"\u0061ps"')
        .replaceFirst('"feedId"', r'"feed\u0049d"');
    expect(
      ReadyNotificationHint.fromJsonBytes(utf8.encode(value)).feedId,
      '019a3f00-1234-7000-8000-000000000001',
    );
  });
  test('field order and ordinary JSON whitespace are accepted', () {
    final value = jsonDecode(_json) as Map<String, dynamic>;
    final reordered = jsonEncode({
      'prosody': value['prosody'],
      'aps': value['aps'],
    });
    expect(
      ReadyNotificationHint.fromJsonBytes(utf8.encode(' \n\t\r$reordered \n'))
          .itemId,
      '019a3f00-1234-7000-8000-000000000002',
    );
  });
  test('4096-byte inclusive boundary accepts legal whitespace padding', () {
    final bytes = [
      ...utf8.encode(_json),
      ...List.filled(4096 - utf8.encode(_json).length, 0x20),
    ];
    expect(bytes.length, 4096);
    expect(
      ReadyNotificationHint.fromJsonBytes(bytes).feedId,
      '019a3f00-1234-7000-8000-000000000001',
    );
  });
  test('oversized input is refused before reading bytes', () {
    expect(
      () => ReadyNotificationHint.fromJsonBytes(_Oversized()),
      throwsFormatException,
    );
  });

  final bad = <String, List<int>>{
    '4097 bytes': [..._golden, ...List.filled(4097 - _golden.length, 0x20)],
    'empty': [],
    'negative byte': [-1, ..._golden],
    'wide byte': [0x100, ..._golden],
    'truncated UTF8': [..._golden, 0xc3],
    'overlong UTF8': [..._golden, 0xc0, 0x80],
    'surrogate UTF8': [..._golden, 0xed, 0xa0, 0x80],
    'BOM': [0xef, 0xbb, 0xbf, ..._golden],
    'duplicate root APS': utf8.encode(
      _json.replaceFirst('{', '{"aps":{"alert":"A new briefing is ready."},'),
    ),
    'duplicate escaped root': utf8.encode(
      _json.replaceFirst(
        '{',
        r'{"\u0061ps":{"alert":"A new briefing is ready."},',
      ),
    ),
    'duplicate root routing': utf8.encode(
      _json.replaceFirst('"prosody":{', '"prosody":{},"prosody":{'),
    ),
    'duplicate alert': utf8.encode(
      _json.replaceFirst(
        '"alert":',
        '"alert":"A new briefing is ready.","alert":',
      ),
    ),
    'duplicate version': utf8.encode(
      _json.replaceFirst('"version":1', '"version":1,"version":1'),
    ),
    'duplicate kind': utf8.encode(
      _json.replaceFirst('"kind":"ready"', '"kind":"ready","kind":"ready"'),
    ),
    'duplicate escaped version': utf8.encode(
      _json.replaceFirst('"version":1', r'"version":1,"\u0076ersion":1'),
    ),
    'duplicate feed': utf8.encode(
      _json.replaceFirst(
        '"feedId":',
        '"feedId":"019a3f00-1234-7000-8000-000000000001","feedId":',
      ),
    ),
    'trailing object': utf8.encode('$_json{}'),
    'trailing garbage': utf8.encode('$_json privateTextNeverRender'),
    'truncated object': utf8.encode(_json.substring(0, _json.length - 1)),
    'trailing comma': utf8.encode('${_json.substring(0, _json.length - 1)},}'),
    'missing colon': utf8.encode(_json.replaceFirst('"aps":', '"aps"')),
    'missing comma': utf8.encode(
      _json.replaceFirst('},"prosody"', '} "prosody"'),
    ),
    'unsupported nested object': utf8.encode(
      _json.replaceFirst(
        '"kind":"ready"',
        '"kind":{"private":"privateTextNeverRender"}',
      ),
    ),
    'unsupported array': utf8.encode(
      _json.replaceFirst('"kind":"ready"', '"kind":["ready"]'),
    ),
    'floating version': utf8.encode(
      _json.replaceFirst('"version":1', '"version":1.0'),
    ),
    'exponent version': utf8.encode(
      _json.replaceFirst('"version":1', '"version":1e0'),
    ),
    'leading zero version': utf8.encode(
      _json.replaceFirst('"version":1', '"version":01'),
    ),
    'string version': utf8.encode(
      _json.replaceFirst('"version":1', '"version":"1"'),
    ),
    'unknown version': utf8.encode(
      _json.replaceFirst('"version":1', '"version":2'),
    ),
    'unknown field': utf8.encode(
      _json.replaceFirst('{', '{"private":"privateTextNeverRender",'),
    ),
    'missing routing': utf8.encode(
      '{"aps":{"alert":"A new briefing is ready."}}',
    ),
    'unescaped control': utf8.encode(_json.replaceFirst('A new', 'A\tnew')),
    'invalid escape': utf8.encode(_json.replaceFirst('ready.', r'ready\x00.')),
    'lone high surrogate': utf8.encode(
      _json.replaceFirst('ready.', r'ready\ud800.'),
    ),
    'lone low surrogate': utf8.encode(
      _json.replaceFirst('ready.', r'ready\udc00.'),
    ),
    'non-JSON whitespace': utf8.encode('\u00a0$_json'),
  };
  for (final entry in bad.entries) {
    test('refuse raw JSON ${entry.key} with fixed private-safe diagnostic', () {
      expect(
        () => ReadyNotificationHint.fromJsonBytes(entry.value),
        throwsA(
          isA<FormatException>()
              .having(
                (e) => e.message,
                'fixed message',
                'Invalid ready notification',
              )
              .having((e) => e.source, 'no private source', isNull)
              .having((e) => e.offset, 'no private offset', isNull),
        ),
      );
    });
  }
}
