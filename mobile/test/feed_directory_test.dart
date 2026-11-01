import 'package:flutter_test/flutter_test.dart';
import '../tool/check_feed_directory.dart';

void main() {
  test(
    'space directory follows bounded pages without hiding failures',
    checkFeedDirectory,
  );
}
