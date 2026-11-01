import 'package:flutter_test/flutter_test.dart';
import '../tool/check_directory_transport.dart';

void main() {
  test(
    'directory and Activity transport preserve scope and reject partial failures',
    checkDirectoryTransport,
  );
}
