import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/data/notification_hint.dart';

Map<String, dynamic> fixture() => jsonDecode(
  File('../testdata/notifications/ready-v1.json').readAsStringSync(),
) as Map<String, dynamic>;

void main() {
  test('Go golden payload decodes to immutable routing hints', () {
    final value = fixture();
    final hint = ReadyNotificationHint.fromPayload(value);
    expect(hint.feedId, '019a3f00-1234-7000-8000-000000000001');
    expect(hint.itemId, '019a3f00-1234-7000-8000-000000000002');
    expect(hint.renditionId, '019a3f00-1234-7000-8000-000000000003');
    (value['prosody'] as Map)['feedId'] = 'changed';
    expect(hint.feedId, '019a3f00-1234-7000-8000-000000000001');
  });
  test('Platform dictionary types preserve the same profile', () {
    final original = fixture();
    final platform = <Object?, Object?>{
      'aps': Map<Object?, Object?>.from(original['aps'] as Map),
      'prosody': Map<Object?, Object?>.from(original['prosody'] as Map),
    };
    expect(
      ReadyNotificationHint.fromPayload(platform).itemId,
      '019a3f00-1234-7000-8000-000000000002',
    );
  });
  test('Existing v4 item and rendition identities remain valid', () {
    final value = fixture();
    final routing = value['prosody'] as Map;
    routing['itemId'] = '550e8400-e29b-41d4-a716-446655440001';
    routing['renditionId'] = '550e8400-e29b-41d4-a716-446655440002';
    final hint = ReadyNotificationHint.fromPayload(value);
    expect(hint.itemId, routing['itemId']);
    expect(hint.renditionId, routing['renditionId']);
  });

  for (final input in <Object?>[null, true, 1, 'private text', [], {}]) {
    test('Refuse non-payload shape ${input.runtimeType}', () {
      expect(
        () => ReadyNotificationHint.fromPayload(input),
        throwsFormatException,
      );
    });
  }
  final malformed = <String, void Function(Map<String, dynamic>)>{
    'missing APS': (p) => p.remove('aps'),
    'missing routing': (p) => p.remove('prosody'),
    'foreign root fields': (p) => p['url'] = 'https://example.invalid/private',
    'non-string platform key': (p) => p['aps'] = {1: 'private'},
    'APS wrong type': (p) => p['aps'] = [],
    'private alert text': (p) => (p['aps'] as Map)['alert'] = 'Private source',
    'rich alert object': (p) =>
        (p['aps'] as Map)['alert'] = {'body': 'private'},
    'unexpected badge': (p) => (p['aps'] as Map)['badge'] = 1,
    'unexpected sound': (p) => (p['aps'] as Map)['sound'] = 'default',
    'routing wrong type': (p) => p['prosody'] = [],
    'unknown version': (p) => (p['prosody'] as Map)['version'] = 2,
    'fractional version': (p) => (p['prosody'] as Map)['version'] = 1.0,
    'boolean version': (p) => (p['prosody'] as Map)['version'] = true,
    'missing version': (p) => (p['prosody'] as Map).remove('version'),
    'foreign kind': (p) => (p['prosody'] as Map)['kind'] = 'submitted',
    'extra private metadata': (p) => (p['prosody'] as Map)['title'] = 'private',
    'extra URL metadata': (p) =>
        (p['prosody'] as Map)['url'] = 'https://example.invalid/private',
    'feed v4': (p) => (p['prosody'] as Map)['feedId'] =
        '550e8400-e29b-41d4-a716-446655440001',
    'feed bad variant': (p) => (p['prosody'] as Map)['feedId'] =
        '019a3f00-1234-7000-0000-000000000001',
  };
  for (final entry in malformed.entries) {
    test('Refuse ${entry.key} with fixed diagnostic', () {
      final input = fixture();
      entry.value(input);
      try {
        ReadyNotificationHint.fromPayload(input);
        fail('Malformed payload accepted');
      } on FormatException catch (error) {
        expect(error.message, 'Invalid ready notification');
        expect(error.source, isNull);
      }
    });
  }
  for (final field in ['feedId', 'itemId', 'renditionId']) {
    test('Refuse absent $field', () {
      final input = fixture();
      (input['prosody'] as Map).remove(field);
      expect(
        () => ReadyNotificationHint.fromPayload(input),
        throwsFormatException,
      );
    });
    for (final entry in <String, Object?>{
      'null': null,
      'non-string': 7,
      'zero': '00000000-0000-0000-0000-000000000000',
      'uppercase': '019A3F00-1234-7000-8000-000000000001',
      'short': '019a3f00',
      'newline': '019a3f00-1234-7000-8000-000000000001\n',
      'URI': 'https://example.invalid/private',
    }.entries) {
      test('Refuse $field ${entry.key}', () {
        final input = fixture();
        (input['prosody'] as Map)[field] = entry.value;
        expect(
          () => ReadyNotificationHint.fromPayload(input),
          throwsFormatException,
        );
      });
    }
  }
}
