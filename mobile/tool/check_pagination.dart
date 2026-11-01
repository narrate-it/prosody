import 'dart:io';
import 'package:prosody/data/pagination.dart';

void checkPagination() {
  void rejects(void Function() action) {
    try {
      action();
    } on FormatException {
      return;
    }
    throw StateError('Invalid pagination accepted');
  }

  final pages = FeedPagination();
  rejects(() => pages.accept(requested: 'unexpected', next: 'a'));
  rejects(() => pages.accept(requested: null, next: ''));
  pages.accept(requested: null, next: 'a');
  rejects(() => pages.accept(requested: 'a', next: 'a'));
  rejects(() => pages.accept(requested: 'wrong', next: 'b'));
  if (pages.next != 'a') throw StateError('Rejected page advanced cursor');
  // A failed transport does not call accept; retry can use the same cursor.
  pages.accept(requested: 'a', next: 'b');
  rejects(() => pages.accept(requested: 'b', next: 'a'));
  if (pages.next != 'b') throw StateError('Cycle changed accepted state');
  pages.accept(requested: 'b', next: 'c');
  rejects(() => pages.accept(requested: 'c', next: 'b'));
  pages.accept(requested: 'c', next: null);
  rejects(() => pages.accept(requested: null, next: 'd'));
  rejects(() => pages.accept(requested: 'c', next: 'd'));
  final empty = FeedPagination();
  empty.accept(requested: null, next: null);
  rejects(() => empty.accept(requested: null, next: 'a'));
  final otherFeed = FeedPagination();
  otherFeed.accept(requested: null, next: 'a');
  otherFeed.accept(requested: 'a', next: null);
}

void main() {
  checkPagination();
  stdout.writeln(
    'Pagination cycle/retry checks passed (no service/UI acceptance).',
  );
}
