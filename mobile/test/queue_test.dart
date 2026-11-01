import 'package:flutter_test/flutter_test.dart';
import '../tool/check_queue.dart';

void main() {
  test('pagination appends unique playable items in order', checkQueue);
}
