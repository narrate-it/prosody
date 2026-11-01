import 'package:flutter_test/flutter_test.dart';
import '../tool/check_activity.dart';

void main() {
  test('activity accepts summaries without playable manifests', checkActivity);
}
