import 'dart:io';
import 'package:prosody/data/activity.dart';

void checkActivity() {
  final summary = <String, dynamic>{
    'id': 'item',
    'title': '',
    'source': 'Original\u0000text',
    'status': 'ready',
    'acceptedAt': '2026-11-01T00:00:00Z',
  };
  for (final status in ['ready', 'processing', 'failed']) {
    final page = ActivityPage.fromJson({
      'items': [
        {...summary, 'status': status},
      ],
    });
    if (page.items.single.source != summary['source'] ||
        page.nextCursor != null) {
      throw StateError('Source or pagination changed');
    }
  }
  void rejects(Map<String, dynamic> value) {
    try {
      ActivityPage.fromJson(value);
    } on FormatException {
      return;
    }
    throw StateError('Malformed activity accepted');
  }

  rejects({
    'items': [
      {...summary, 'status': 'deleted'},
    ],
  });
  rejects({
    'items': [
      {...summary, 'acceptedAt': 'not-time'},
    ],
  });
  rejects({
    'items': [
      {...summary, 'acceptedAt': '2026-11-01T00:00:00'},
    ],
  });
  rejects({
    'items': [
      {...summary, 'source': null},
    ],
  });
  rejects({
    'items': [null],
  });
  rejects({'items': [], 'nextCursor': ''});
  rejects({'items': [], 'nextCursor': 1});
  if (ActivityPage.fromJson({'items': []}).items.isNotEmpty) {
    throw StateError('Empty activity changed');
  }
}

void main() {
  checkActivity();
  stdout.writeln(
    'Activity summary contract passed; no HTTP/widget acceptance.',
  );
}
