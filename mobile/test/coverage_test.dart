import 'package:flutter_test/flutter_test.dart';
import '../tool/check_coverage.dart';

void main() {
  test('played coverage counts only valid continuous progress', checkCoverage);
}
