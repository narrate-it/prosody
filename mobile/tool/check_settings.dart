import 'dart:io';
import 'package:prosody/data/models.dart';

void checkSettings() {
  final initial = <String, dynamic>{
    'revision': 0,
    'hasKey': false,
    'notifications': false,
  };
  final empty = FeedSettings.fromJson(initial);
  if (empty.revision != 0 ||
      empty.textModel != null ||
      empty.hasKey ||
      empty.notifications) {
    throw StateError('Initial settings changed');
  }
  final configured = {
    ...initial,
    'revision': 42,
    'hasKey': true,
    'textModel': 'text',
    'speechModel': 'speech',
    'voice': 'voice',
  };
  final valid = FeedSettings.fromJson(configured);
  if (valid.revision != 42 || valid.voice != 'voice' || !valid.hasKey) {
    throw StateError('Configured settings changed');
  }
  void rejects(Map<String, dynamic> value) {
    try {
      FeedSettings.fromJson(value);
    } on FormatException {
      return;
    }
    throw StateError('Invalid settings accepted');
  }

  for (final revision in [-1, 0.5, '1', null, 9007199254740992]) {
    rejects({...initial, 'revision': revision});
  }
  for (final field in ['hasKey', 'notifications']) {
    for (final value in [null, 0, 'true']) {
      rejects({...initial, field: value});
    }
  }
  for (final field in ['textModel', 'speechModel', 'voice']) {
    rejects({...configured, field: ' '});
    rejects({...configured, field: 7});
  }
}

void main() {
  checkSettings();
  stdout.writeln('Settings response checks passed; no API/widget acceptance.');
}
