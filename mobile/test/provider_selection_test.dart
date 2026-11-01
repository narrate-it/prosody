import 'package:flutter_test/flutter_test.dart';
import '../tool/check_provider_selection.dart';

void main() {
  test(
    'provider selection requires current models, voice and key',
    checkProviderSelection,
  );
}
