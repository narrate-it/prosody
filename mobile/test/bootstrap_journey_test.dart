import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:just_audio/just_audio.dart' as audio;
import 'package:prosody/data/native_session.dart';
import 'package:prosody/main.dart' as app;
import 'package:prosody/playback/handler.dart';
import 'package:prosody/ui/feed.dart';

class _Audio extends Fake implements audio.AudioPlayer {}

class _Playback extends ProsodyPlayback {
  _Playback() : super(audioPlayer: _Audio());
  bool disposed = false;

  @override
  Future<void> dispose() async {
    disposed = true;
    location.dispose();
    video.dispose();
    failure.dispose();
    discontinuity.dispose();
    completed.dispose();
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const bridge = MethodChannel('run.prosody/native_session');
  const origin = 'https://api.example.invalid';
  const issuer = 'https://identity.example.invalid';
  const audience = '$origin/';
  late List<MethodCall> calls;

  setUp(() {
    calls = [];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, (call) async {
          calls.add(call);
          return null;
        });
  });
  tearDown(
    () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, null),
  );

  app.ProsodyApp configured(_Playback playback) =>
      app.ProsodyApp.fromReleaseInputs(
        playback: playback,
        origin: origin,
        issuer: issuer,
        audience: audience,
      );

  FilledButton button(WidgetTester tester) =>
      tester.widget<FilledButton>(find.byType(FilledButton));

  void portrait(WidgetTester tester) {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

  testWidgets(
    'default entry-point composition stays disabled with no platform IO',
    (tester) async {
      portrait(tester);
      final playback = _Playback();
      await tester.pumpWidget(
        app.ProsodyApp.fromReleaseInputs(playback: playback),
      );
      await tester.pumpAndSettle();
      expect(button(tester).onPressed, isNull);
      expect(
        find.text('Sign-in is not available in this build.'),
        findsOneWidget,
      );
      expect(find.byType(FeedScreen), findsNothing);
      expect(calls, isEmpty);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      expect(playback.disposed, isTrue);
    },
  );

  for (final missing in ['origin', 'issuer', 'audience']) {
    testWidgets('partial $missing input cannot enable entry-point sign-in', (
      tester,
    ) async {
      portrait(tester);
      await tester.pumpWidget(
        app.ProsodyApp.fromReleaseInputs(
          playback: _Playback(),
          origin: missing == 'origin' ? '' : origin,
          issuer: missing == 'issuer' ? '' : issuer,
          audience: missing == 'audience' ? '' : audience,
        ),
      );
      await tester.pumpAndSettle();
      expect(button(tester).onPressed, isNull);
      expect(calls, isEmpty);
      expect(find.byType(FeedScreen), findsNothing);
    });
  }

  testWidgets('malformed release input stays disabled without leaking it', (
    tester,
  ) async {
    portrait(tester);
    await tester.pumpWidget(
      app.ProsodyApp.fromReleaseInputs(
        playback: _Playback(),
        origin: origin,
        issuer: issuer,
        audience: '$origin/PRIVATE?token=PRIVATE',
      ),
    );
    await tester.pumpAndSettle();
    expect(button(tester).onPressed, isNull);
    expect(find.textContaining('PRIVATE'), findsNothing);
    expect(calls, isEmpty);
  });

  testWidgets(
    'complete binding restores signed-out state without opening browser',
    (tester) async {
      portrait(tester);
      final widget = configured(_Playback());
      expect(widget.session, isA<NativeSession>());
      expect(widget.origin.toString(), origin);
      await tester.pumpWidget(widget);
      await tester.pumpAndSettle();
      expect(button(tester).onPressed, isNotNull);
      expect(calls.map((c) => c.method), ['cancel', 'read', 'read']);
      expect(calls.where((c) => c.method == 'authenticate'), isEmpty);
      expect(find.byType(FeedScreen), findsNothing);
    },
  );

  testWidgets('pending recovery cannot render authenticated feed on startup', (
    tester,
  ) async {
    portrait(tester);
    final widget = configured(_Playback());
    final session = widget.session! as NativeSession;
    final active = session.configuration.binding;
    final records = <String, Map<String, String>>{
      active: {'token': 'a' * 43, 'expires': '2027-01-01T00:00:00Z'},
      jsonEncode([active, 'pending-revocation.v1']): {
        'token': 'a' * 43,
        'expires': '2027-01-01T00:00:00Z',
      },
    };
    // Exact pending record identity is separately bound by the native client.
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, (call) async {
          calls.add(call);
          if (call.method == 'cancel') {
            return null;
          }
          final binding = (call.arguments as Map)['binding'] as String;
          if (call.method == 'read') {
            return records[binding];
          }
          if (call.method == 'write') {
            records[binding] = {
              'token': (call.arguments as Map)['token'] as String,
              'expires': (call.arguments as Map)['expires'] as String,
            };
          }
          if (call.method == 'delete') {
            records.remove(binding);
          }
          return null;
        });
    await tester.pumpWidget(widget);
    await tester.pumpAndSettle();
    expect(find.byType(FeedScreen), findsNothing);
    expect(button(tester).onPressed, isNotNull);
    expect(records.containsKey(active), isFalse);
    expect(records.values.single['token'], 'a' * 43);
    expect(calls.where((c) => c.method == 'authenticate'), isEmpty);
    await expectLater(
      session.accessToken(),
      throwsA(isA<NativeSessionFailure>()),
    );
  });

  testWidgets('Keychain failure stays generic and cannot open browser', (
    tester,
  ) async {
    portrait(tester);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(bridge, (call) async {
          calls.add(call);
          if (call.method == 'read') {
            throw PlatformException(code: 'PRIVATE', message: 'PRIVATE');
          }
          return null;
        });
    await tester.pumpWidget(configured(_Playback()));
    await tester.pumpAndSettle();
    expect(
      find.text('This build cannot connect to Prosody yet.'),
      findsOneWidget,
    );
    expect(find.textContaining('PRIVATE'), findsNothing);
    expect(find.byType(FeedScreen), findsNothing);
    await tester.tap(find.byType(FilledButton));
    await tester.pumpAndSettle();
    expect(
      find.text('Sign-in did not finish. Please try again.'),
      findsOneWidget,
    );
    expect(calls.where((c) => c.method == 'authenticate'), isEmpty);
    expect(find.byType(FeedScreen), findsNothing);
  });

  testWidgets(
    'cancelled browser login stays signed out and permits explicit retry',
    (tester) async {
      portrait(tester);
      final urls = <Uri>[];
      var proofs = 0;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(bridge, (call) async {
            calls.add(call);
            if (call.method == 'proof') {
              proofs++;
              return {
                'verifier': 'v' * 43,
                'state': (proofs == 1 ? 's' : 't') * 43,
                'challenge': 'c' * 43,
              };
            }
            if (call.method == 'authenticate') {
              urls.add(Uri.parse((call.arguments as Map)['url'] as String));
              throw PlatformException(code: 'cancelled', message: 'PRIVATE');
            }
            return null;
          });
      await tester.pumpWidget(configured(_Playback()));
      await tester.pumpAndSettle();
      expect(urls, isEmpty);
      for (var attempt = 0; attempt < 2; attempt++) {
        await tester.tap(find.byType(FilledButton));
        await tester.pumpAndSettle();
        expect(urls.length, attempt + 1);
        expect(button(tester).onPressed, isNotNull);
        expect(
          find.text('Sign-in did not finish. Please try again.'),
          findsOneWidget,
        );
        expect(find.textContaining('PRIVATE'), findsNothing);
        expect(find.byType(FeedScreen), findsNothing);
      }
      expect(urls[0].origin, issuer);
      expect(urls[0].path, '/oauth/authorize');
      expect(
        urls[0].queryParameters['client_id'],
        NativeSessionConfiguration.clientId,
      );
      expect(
        urls[0].queryParameters['redirect_uri'],
        NativeSessionConfiguration.redirect,
      );
      expect(urls[0].queryParameters['code_challenge_method'], 'S256');
      expect(urls[0].queryParameters['scope'], 'personal');
      expect(
        urls[0].queryParameters['state'],
        isNot(urls[1].queryParameters['state']),
      );
    },
  );
}
