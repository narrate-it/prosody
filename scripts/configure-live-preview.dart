// Provision a debug preview over its existing authenticated localhost VM tunnel.
// Never include the API key in command arguments, Flutter defines or assets.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

Future<void> main(List<String> args) async {
  WebSocket? socket;
  StreamSubscription<dynamic>? subscription;
  try {
    if (args.length != 2) throw StateError('usage');
    final service = Uri.parse(args[0]);
    if (service.scheme != 'http' ||
        service.host != '127.0.0.1' ||
        service.userInfo.isNotEmpty ||
        service.hasQuery ||
        service.hasFragment ||
        service.path == '/')
      throw StateError('localhost service required');
    String? key;
    for (var line in await File(args[1]).readAsLines()) {
      line = line.trim();
      if (line.startsWith('export ')) line = line.substring(7).trim();
      if (!line.startsWith('OPENROUTER_API_KEY=')) continue;
      var value = line.substring('OPENROUTER_API_KEY='.length).trim();
      if ((value.startsWith('"') && value.endsWith('"')) ||
          (value.startsWith("'") && value.endsWith("'"))) {
        value = value.substring(1, value.length - 1);
      }
      key = value;
    }
    if (key == null || key.isEmpty || key.contains(RegExp(r'\s')))
      throw StateError('key required');
    socket = await WebSocket.connect(
      service
          .replace(
            scheme: 'ws',
            path:
                '${service.path.endsWith('/') ? service.path : '${service.path}/'}ws',
          )
          .toString(),
    ).timeout(const Duration(seconds: 15));
    final pending = <String, Completer<Map<String, dynamic>>>{};
    void failPending() {
      for (final reply in pending.values) {
        if (!reply.isCompleted)
          reply.completeError(StateError('Debug connection closed'));
      }
      pending.clear();
    }

    subscription = socket.listen(
      (event) {
        try {
          final value = jsonDecode(event as String) as Map<String, dynamic>;
          pending.remove(value['id'])?.complete(value);
        } catch (_) {
          failPending();
        }
      },
      onError: (Object _) => failPending(),
      onDone: failPending,
    );
    var sequence = 0;
    Future<Map<String, dynamic>> rpc(
      String method, [
      Map<String, Object?>? params,
    ]) async {
      final id = '${++sequence}';
      final reply = Completer<Map<String, dynamic>>();
      pending[id] = reply;
      socket!.add(
        jsonEncode({
          'jsonrpc': '2.0',
          'id': id,
          'method': method,
          'params': params ?? {},
        }),
      );
      final value = await reply.future.timeout(const Duration(seconds: 15));
      if (value.containsKey('error'))
        throw StateError('debug configuration rejected');
      return value['result'] as Map<String, dynamic>;
    }

    final vm = await rpc('getVM');
    final isolate = (vm['isolates'] as List)
        .cast<Map<String, dynamic>>()
        .firstWhere((i) => i['name'] == 'main');
    final result = await rpc('ext.prosody.configureLivePreview', {
      'isolateId': isolate['id'],
      'sessionId': 'prosody-live-20261101-1',
      'key': key,
    });
    if (result['configured'] != true)
      throw StateError('configuration not confirmed');
    stdout.writeln(
      'Live preview configured; remaining reserved attempts: ${result['remainingAttempts']}.',
    );
  } catch (_) {
    stderr.writeln(
      'Could not configure live preview. Use the attached localhost VM service URL and the local .env path; no key was logged.',
    );
    exitCode = 1;
  } finally {
    await subscription?.cancel();
    await socket?.close();
  }
}
