import 'package:flutter_test/flutter_test.dart';
import '../tool/check_bounded_body.dart';

void main() {
  test(
    'response collection has absolute bounds and cancellation',
    checkBoundedBody,
  );
}
