import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:prosody/preview/live_provider.dart';
import 'package:prosody/preview/live_session.dart';

class _Provider implements LiveBriefingProvider {
  int calls = 0, dispatched = 0;
  bool fail = false;
  Completer<void>? hold;
  Future<void> Function()? afterReservation;

  @override
  Future<LiveBriefing> generate({
    required String key,
    required String source,
    required Future<void> Function() reserveAttempt,
  }) async {
    calls++;
    await reserveAttempt();
    dispatched++;
    await afterReservation?.call();
    await hold?.future;
    if (fail) {
      throw const LiveProviderFailure(
        'private provider detail',
        outcomeUnknown: true,
      );
    }
    return LiveBriefing(
      source: source,
      script: 'A briefing.',
      mp3: Uint8List(3),
    );
  }

  @override
  void close() {}
}

void main() {
  late Directory directory;
  late File ledger;
  setUp(() async {
    directory = await Directory.systemTemp.createTemp(
      'prosody-live-session-test-',
    );
    ledger = File('${directory.path}/budget.jsonl');
  });
  tearDown(() async => directory.delete(recursive: true));

  LiveSession session(_Provider client) {
    final value = LiveSession(ledger: ledger, client: client);
    addTearDown(value.dispose);
    return value;
  }

  test(
    'flushes reservation before dispatch, restart retains budget but no key',
    () async {
      final client = _Provider();
      final value = session(client);
      await value.initialize();
      expect(value.configured, isFalse);
      value.configure('synthetic-private-key');
      client.afterReservation = () async {
        final records = (await ledger.readAsLines()).map(jsonDecode).toList();
        expect(records.last['attempts'], 1);
      };
      final result = await value.generate('Private source message');
      expect(result.script, 'A briefing.');
      expect(value.remainingAttempts, 3);
      expect(value.remainingBudgetDollars, 0.75);
      final persisted = await ledger.readAsString();
      expect(persisted, isNot(contains('synthetic-private-key')));
      expect(persisted, isNot(contains('Private source')));
      final restarted = session(_Provider());
      await restarted.initialize();
      expect(restarted.remainingAttempts, 3);
      expect(restarted.configured, isFalse);
    },
  );

  test(
    'unknown outcomes consume all four attempts without retry or refund',
    () async {
      final client = _Provider()..fail = true;
      final value = session(client);
      await value.initialize();
      value.configure('synthetic-key');
      for (var i = 0; i < 4; i++) {
        await expectLater(
          value.generate('Message'),
          throwsA(isA<LiveSessionFailure>()),
        );
        expect(value.remainingAttempts, 3 - i);
        expect(value.error, isNot(contains('private provider detail')));
      }
      await expectLater(
        value.generate('Message'),
        throwsA(isA<LiveSessionFailure>()),
      );
      expect(client.calls, 4);
      expect(client.dispatched, 4);
      value.clearKey();
      value.configure('another-synthetic-key');
      expect(value.remainingAttempts, 0);
      final restarted = session(_Provider());
      await restarted.initialize();
      expect(restarted.remainingAttempts, 0);
    },
  );

  test(
    'malformed, truncated, empty and wrong-session ledgers fail closed',
    () async {
      for (final content in [
        '',
        'not json\n',
        '{"version":1,"sessionId":"${LiveSession.sessionId}","attempts":0}',
        '{"version":1,"sessionId":"another-session","attempts":0}\n',
        '{"version":1,"sessionId":"${LiveSession.sessionId}","attempts":2}\n',
      ]) {
        await ledger.writeAsString(content, flush: true);
        final client = _Provider();
        final value = session(client);
        await value.initialize();
        value.configure('synthetic-key');
        expect(value.configured, isFalse);
        expect(value.remainingAttempts, 0);
        await expectLater(
          value.generate('Message'),
          throwsA(isA<LiveSessionFailure>()),
        );
        expect(client.calls, 0);
        expect(await ledger.readAsString(), content);
      }
    },
  );

  test(
    'one session rejects overlapping generation without second reservation',
    () async {
      final reserved = Completer<void>();
      final client = _Provider()
        ..hold = Completer<void>()
        ..afterReservation = () async {
          reserved.complete();
        };
      final value = session(client);
      await value.initialize();
      value.configure('synthetic-key');
      final first = value.generate('First');
      await reserved.future;
      expect(value.busy, isTrue);
      await expectLater(
        value.generate('Second'),
        throwsA(isA<LiveSessionFailure>()),
      );
      expect(client.calls, 1);
      client.hold!.complete();
      await first;
      expect(value.busy, isFalse);
      expect(value.remainingAttempts, 3);
    },
  );

  test(
    'separate session instances serialize shared ledger reservations',
    () async {
      final first = session(_Provider());
      final second = session(_Provider());
      await Future.wait([first.initialize(), second.initialize()]);
      first.configure('synthetic-one');
      second.configure('synthetic-two');
      await Future.wait([first.generate('One'), second.generate('Two')]);
      final restarted = session(_Provider());
      await restarted.initialize();
      expect(restarted.remainingAttempts, 2);
      final records = (await ledger.readAsLines()).map(jsonDecode).toList();
      expect(records.map((record) => record['attempts']), [0, 1, 2]);
    },
  );

  test(
    'disappeared ledger blocks paid dispatch and cannot reset allowance',
    () async {
      final client = _Provider();
      final value = session(client);
      await value.initialize();
      value.configure('synthetic-key');
      await ledger.delete();
      await expectLater(
        value.generate('Message'),
        throwsA(isA<LiveSessionFailure>()),
      );
      expect(client.dispatched, 0);
      expect(value.configured, isFalse);
      expect(value.remainingAttempts, 0);
      expect(await ledger.exists(), isFalse);
    },
  );

  test('invalid source or absent key cannot reserve; clearing key preserves budget', () async {
    final client = _Provider();
    final value = session(client);
    await value.initialize();
    for (final key in ['', '  ', 'synthetic\nkey', 'synthetic\rkey']) {
      expect(() => value.configure(key), throwsA(isA<LiveSessionFailure>()));
    }
    await expectLater(
      value.generate('Message'),
      throwsA(isA<LiveSessionFailure>()),
    );
    value.configure('synthetic-key');
    for (final source in ['', '  ', 'a' * 4001]) {
      await expectLater(
        value.generate(source),
        throwsA(isA<LiveSessionFailure>()),
      );
    }
    expect(client.calls, 0);
    expect(value.remainingAttempts, 4);
    value.clearKey();
    await expectLater(
      value.generate('Message'),
      throwsA(isA<LiveSessionFailure>()),
    );
    expect(value.remainingAttempts, 4);
  });
}
