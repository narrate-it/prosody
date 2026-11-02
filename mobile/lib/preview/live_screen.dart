import 'package:flutter/material.dart';

import 'live_provider.dart';
import 'live_session.dart';

class LiveBriefingScreen extends StatefulWidget {
  final LiveSession session;
  final Future<void> Function(LiveBriefing) onGenerated;
  const LiveBriefingScreen({
    super.key,
    required this.session,
    required this.onGenerated,
  });
  @override
  State<LiveBriefingScreen> createState() => _LiveBriefingScreenState();
}

class _LiveBriefingScreenState extends State<LiveBriefingScreen> {
  final _text = TextEditingController();
  bool _submitting = false;
  String? _error;

  Future<void> _generate() async {
    if (_submitting ||
        widget.session.busy ||
        !widget.session.configured ||
        widget.session.remainingAttempts == 0) {
      return;
    }
    setState(() {
      _submitting = true;
      _error = null;
    });
    var generated = false;
    try {
      final briefing = await widget.session.generate(_text.text);
      generated = true;
      // The session and destination outlive this route. Leaving the editor
      // must not discard audio from an already reserved provider request.
      await widget.onGenerated(briefing);
      if (mounted && ModalRoute.of(context)?.isCurrent == true) {
        Navigator.of(context).pop();
      }
    } catch (_) {
      if (mounted) {
        setState(
          () => _error = generated
              ? 'The briefing was generated, but playback could not be opened. This attempt remains reserved.'
              : widget.session.error ?? 'Generation could not start. Check your text and preview setup.',
        );
      }
    } finally {
      if (mounted) {
        setState(() => _submitting = false);
      }
    }
  }

  @override
  void dispose() {
    _text.dispose();
    // The caller owns the session and durable allowance. Leaving this screen
    // must not reset reservations or start another request.
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Create live briefing')),
    body: SafeArea(
      child: ListenableBuilder(
        listenable: widget.session,
        builder: (context, _) {
          final session = widget.session;
          final busy = _submitting || session.busy;
          return SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Turn a message into a briefing.',
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 12),
                const Text(
                  'Live preview · GPT-4.1 mini\nElevenLabs v4 Turbo · Laura',
                ),
                const SizedBox(height: 16),
                const Text(
                  'This uses the key provisioned for this preview. No production account or agent connection is required.',
                ),
                const SizedBox(height: 20),
                TextField(
                  controller: _text,
                  enabled: !busy,
                  minLines: 5,
                  maxLines: 10,
                  maxLength: 4000,
                  keyboardType: TextInputType.multiline,
                  decoration: const InputDecoration(
                    labelText: 'Message to narrate',
                    hintText: 'Paste an update you would like to listen to.',
                  ),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Maximum reserved per attempt: \$0.25. The total authorized preview allowance is \$1.00. Failed or uncertain attempts remain reserved.',
                ),
                const SizedBox(height: 8),
                Text(
                  '\$${session.remainingBudgetDollars.toStringAsFixed(2)} remaining · ${session.remainingAttempts} attempts',
                ),
                if (!session.configured) ...[
                  const SizedBox(height: 12),
                  const Text(
                    'Live generation is not configured for this preview.',
                  ),
                ],
                if (_error != null || session.error != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    _error ?? session.error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ],
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed:
                        !session.configured ||
                            session.remainingAttempts == 0 ||
                            busy
                        ? null
                        : _generate,
                    child: Text(busy ? 'Generating…' : 'Generate briefing'),
                  ),
                ),
              ],
            ),
          );
        },
      ),
    ),
  );
}
