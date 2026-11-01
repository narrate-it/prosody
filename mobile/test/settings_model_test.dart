import 'package:flutter_test/flutter_test.dart';
import '../tool/check_settings.dart';

void main() {
  test(
    'settings preserve exact revisions and explicit preferences',
    checkSettings,
  );
}
