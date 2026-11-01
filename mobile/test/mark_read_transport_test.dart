import 'package:flutter_test/flutter_test.dart';
import '../tool/check_mark_read_transport.dart';

void main() {
  test(
    'mark-read transport preserves revisions and never retries uncertain writes',
    checkMarkReadTransport,
  );
}
