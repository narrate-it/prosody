import 'dart:io';
import 'package:prosody/data/models.dart';
import 'package:prosody/data/provider_selection.dart';

void checkProviderSelection() {
  const text = [ProviderModel('text', 'Text', [])];
  const speech = [
    ProviderModel('speech', 'Speech', ['voice']),
  ];
  String? validate({
    String? model = 'text',
    String? audio = 'speech',
    String? voice = 'voice',
    bool hasKey = true,
    String key = '',
    List<ProviderModel> texts = text,
    List<ProviderModel> speeches = speech,
  }) => providerSelectionError(
    textModels: texts,
    speechModels: speeches,
    textModel: model,
    speechModel: audio,
    voice: voice,
    hasKey: hasKey,
    replacementKey: key,
  );
  void expect(bool value) {
    if (!value) throw StateError('Selection check failed');
  }

  expect(validate() == null);
  expect(validate(hasKey: false, key: 'test-key') == null);
  expect(validate(key: 'replacement') == null);
  expect(validate(hasKey: false) != null);
  expect(validate(hasKey: false, key: '  ') != null);
  expect(validate(key: '  ') != null);
  expect(validate(model: null) != null);
  expect(validate(audio: null) != null);
  expect(validate(voice: null) != null);
  expect(validate(model: 'retired') != null);
  expect(validate(audio: 'retired') != null);
  expect(validate(voice: 'retired') != null);
  expect(validate(texts: []) != null);
  expect(validate(speeches: []) != null);
  expect(validate(texts: [...text, ...text]) != null);
  expect(validate(speeches: [...speech, ...speech]) != null);
  expect(
    validate(speeches: [const ProviderModel('speech', 'Speech', [])]) != null,
  );
}

void main() {
  checkProviderSelection();
  stdout.writeln(
    'Provider selection checks passed (no provider/UI acceptance).',
  );
}
