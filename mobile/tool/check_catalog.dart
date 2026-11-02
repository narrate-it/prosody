import 'dart:io';
import 'package:prosody/data/models.dart';

void checkCatalog() {
  final model = <String, dynamic>{
    'id': 'speech',
    'name': 'Speech',
    'voices': ['voice'],
  };
  void rejects(Object? value) {
    try {
      providerCatalogFromJson(value);
    } on FormatException {
      return;
    }
    throw StateError('Invalid catalog accepted');
  }

  final decoded = providerCatalogFromJson([model]);
  if (decoded.single.id != 'speech' ||
      decoded.single.voices.single != 'voice') {
    throw StateError('Valid selection changed');
  }
  (model['voices'] as List).add('later');
  if (decoded.single.voices.length != 1) {
    throw StateError('Catalog aliases input');
  }
  rejects([model, model]);
  rejects([
    {
      'id': 'text',
      'name': 'Text',
      'voices': ['same', 'same'],
    },
  ]);
  rejects([
    {'id': '', 'name': 'Text'},
  ]);
  rejects([
    {'id': 'text', 'name': ' '},
  ]);
  rejects([
    {
      'id': 'text',
      'name': 'Text',
      'voices': [' '],
    },
  ]);
  rejects([
    {
      'id': 'text',
      'name': 'Text',
      'voices': [1],
    },
  ]);
  rejects([
    {'id': 'text', 'name': 'Text', 'voices': 'voice'},
  ]);
  rejects([null]);
  rejects(null);
  if (providerCatalogFromJson([]).isNotEmpty) {
    throw StateError('Empty catalog changed');
  }
  final text = providerCatalogFromJson([
    {'id': 'text', 'name': 'Text'},
  ]);
  if (text.single.voices.isNotEmpty) throw StateError('Invented voices');
}

void main() {
  checkCatalog();
  stdout.writeln(
    'Catalog validation passed; no provider or widget acceptance.',
  );
}
