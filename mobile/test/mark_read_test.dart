import 'package:flutter_test/flutter_test.dart';
import '../tool/check_mark_read.dart';

void main() {
  test(
    'explicit mark-read preserves resume position and coverage',
    checkMarkRead,
  );
}
