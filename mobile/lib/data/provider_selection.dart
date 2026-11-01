import 'models.dart';

// Client-side guidance only. The service must independently validate current
// catalog capabilities, caller authority, revision and credential admission.
String? providerSelectionError({
  required List<ProviderModel> textModels,
  required List<ProviderModel> speechModels,
  required String? textModel,
  required String? speechModel,
  required String? voice,
  required bool hasKey,
  required String replacementKey,
}) {
  if (textModel == null || speechModel == null || voice == null) {
    return 'Choose a briefing model, speech model and voice.';
  }
  if (textModels.where((m) => m.id == textModel).length != 1) {
    return 'Choose an available briefing model.';
  }
  final speech = speechModels.where((m) => m.id == speechModel).toList();
  if (speech.length != 1 || !speech.single.voices.contains(voice)) {
    return 'Choose an available speech model and voice.';
  }
  if ((!hasKey || replacementKey.isNotEmpty) && replacementKey.trim().isEmpty) {
    return 'Enter an OpenRouter key.';
  }
  return null;
}
