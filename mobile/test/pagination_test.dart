import 'package:flutter_test/flutter_test.dart';
import '../tool/check_pagination.dart';

void main() {
  test('pagination rejects cycles without advancing state', checkPagination);
}
