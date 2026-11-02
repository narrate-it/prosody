import 'package:flutter/material.dart';

class DeleteBriefingButton extends StatefulWidget {
  final Future<void> Function() onDelete;
  const DeleteBriefingButton({super.key, required this.onDelete});
  @override
  State<DeleteBriefingButton> createState() => _DeleteBriefingButtonState();
}

class _DeleteBriefingButtonState extends State<DeleteBriefingButton> {
  bool _busy = false;
  String? _error;

  Future<void> _delete() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Delete briefing?'),
          content: const Text(
            'Remove this briefing from your inbox. This cannot be undone.',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Delete'),
            ),
          ],
        ),
      );
      if (!mounted || confirmed != true) return;
      await widget.onDelete();
      if (mounted && ModalRoute.of(context)?.isCurrent == true) {
        Navigator.pop(context);
      }
    } catch (_) {
      if (mounted) {
        setState(() => _error = 'Could not delete this briefing. Try again.');
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    children: [
      TextButton.icon(
        onPressed: _busy ? null : _delete,
        icon: const Icon(Icons.delete_outline),
        label: Text(_busy ? 'Deleting…' : 'Delete briefing'),
      ),
      if (_error != null)
        Text(
          _error!,
          style: TextStyle(color: Theme.of(context).colorScheme.error),
        ),
    ],
  );
}
