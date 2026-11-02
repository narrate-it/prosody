import 'dart:io';
import 'package:prosody/data/feed_directory.dart';

Future<void> checkFeedDirectory() async {
  Map<String, dynamic> feed(String id) => {
    'id': id,
    'name': id,
    'kind': 'team',
    'canManage': false,
  };
  final cursors = <String?>[];
  final result = await loadFeedDirectory((cursor) async {
    cursors.add(cursor);
    return cursor == null
        ? {
            'feeds': [feed('first')],
            'nextCursor': 'next',
          }
        : {
            'feeds': [feed('second')],
          };
  });
  if (result.length != 2 ||
      result.last.id != 'second' ||
      cursors.toString() != '[null, next]') {
    throw StateError('Paginated team space lost');
  }
  Future<void> rejects(Future<Object?> Function() action) async {
    try {
      await action();
    } on FormatException {
      return;
    }
    throw StateError('Invalid directory accepted');
  }

  await rejects(
    () => loadFeedDirectory((_) async => {'feeds': [], 'nextCursor': 'cycle'}),
  );
  await rejects(
    () => loadFeedDirectory(
      (_) async => {
        'feeds': [feed('same'), feed('same')],
      },
    ),
  );
  await rejects(
    () => loadFeedDirectory(
      (_) async => {
        'feeds': [feed('one'), feed('two')],
      },
      maxFeeds: 1,
    ),
  );
  var pages = 0;
  await rejects(
    () => loadFeedDirectory(
      (_) async => {'feeds': [], 'nextCursor': '${++pages}'},
      maxPages: 2,
    ),
  );
  if (pages != 2) throw StateError('Page ceiling exceeded');
  await rejects(
    () => loadFeedDirectory(
      (_) async => {
        'feeds': [
          {...feed('bad'), 'canManage': null},
        ],
      },
    ),
  );
  await rejects(
    () => loadFeedDirectory(
      (_) async => {
        'feeds': [null],
      },
    ),
  );
  final empty = await loadFeedDirectory((_) async => {'feeds': []});
  if (empty.isNotEmpty) throw StateError('Empty directory changed');
  final failure = StateError('page failed');
  var attempts = 0;
  try {
    await loadFeedDirectory((cursor) async {
      attempts++;
      if (cursor == null) {
        return {
          'feeds': [feed('one')],
          'nextCursor': 'two',
        };
      }
      throw failure;
    });
    throw StateError('Partial directory returned after failure');
  } catch (error) {
    if (!identical(error, failure) || attempts != 2) rethrow;
  }
}

Future<void> main() async {
  await checkFeedDirectory();
  stdout.writeln('Feed directory pagination passed; no HTTP/UI acceptance.');
}
