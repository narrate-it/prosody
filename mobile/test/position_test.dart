import 'package:flutter_test/flutter_test.dart';
import '../tool/check_position.dart';

void main() {
  test(
    'paused seeks update resume position without listened coverage',
    checkPosition,
  );
}
