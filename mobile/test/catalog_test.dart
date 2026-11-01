import 'package:flutter_test/flutter_test.dart';
import '../tool/check_catalog.dart';

void main() {
  test('catalog rejects ambiguous model and voice identities', checkCatalog);
}
