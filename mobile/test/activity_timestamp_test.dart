import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/data/activity.dart';

ActivitySummary summaryAt(String value) => ActivitySummary.fromJson({
  'id': 'item',
  'title': 'Briefing',
  'source': 'Original',
  'status': 'ready',
  'acceptedAt': value,
});

void main() {
  for (final value in [
    '2026-02-30T00:00:00Z',
    '2025-02-29T00:00:00Z',
    '1900-02-29T00:00:00Z',
    '2026-04-31T00:00:00Z',
    '2026-00-01T00:00:00Z',
    '2026-13-01T00:00:00Z',
    '2026-01-00T00:00:00Z',
    '2026-01-32T00:00:00Z',
    '2026-01-01T24:00:00Z',
    '2026-01-01T00:60:00Z',
    '2026-01-01T00:00:60Z',
    '2026-01-01T00:00:00+24:00',
    '2026-01-01T00:00:00+00:60',
    '2026-01-01T00:00:00-00:60',
    '2026-01-01 00:00:00Z',
    '20260101T000000Z',
    '2026-01-01T00:00:00Z\n',
    '2026-01-01T00:00:00',
  ]) {
    test('Reject malformed acceptedAt $value', () {
      expect(() => summaryAt(value), throwsFormatException);
    });
  }

  for (final value in [
    '2026-01-01T00:00:00Z',
    '2024-02-29T23:59:59Z',
    '2000-02-29T23:59:59.123456789Z',
    '2026-04-30T00:00:00+00:00',
    '2026-12-31T23:59:59-08:00',
    '2026-01-01T00:00:00+05:30',
  ]) {
    test('Retain valid acceptedAt instant $value', () {
      final result = summaryAt(value);
      expect(result.acceptedAt, DateTime.parse(value));
      expect(result.acceptedAt.isUtc, isTrue);
      expect(result.source, 'Original');
    });
  }
}
