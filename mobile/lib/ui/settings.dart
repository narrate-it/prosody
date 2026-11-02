import 'dart:async';
import 'package:flutter/material.dart';
import '../data/api.dart';
import '../data/models.dart';
import '../data/provider_selection.dart';

class SettingsScreen extends StatefulWidget {
  final ProsodyApi api;
  final Feed feed;
  const SettingsScreen({super.key, required this.api, required this.feed});
  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _key = TextEditingController();
  FeedSettings? _settings;
  List<ProviderModel> _text = [], _speech = [];
  String? _textId, _speechId, _voice, _error;
  bool _saving = false;
  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final settings = await widget.api.settings(widget.feed.id);
      final text = await widget.api.models('text');
      final speech = await widget.api.models('speech');
      if (!mounted) return;
      setState(() {
        _settings = settings;
        _text = text;
        _speech = speech;
        _textId = settings.textModel;
        _speechId = settings.speechModel;
        _voice = settings.voice;
        _error = null;
      });
    } catch (e) {
      if (mounted) {
        setState(
          () => _error = e is ApiFailure
              ? e.message
              : 'Settings could not be loaded.',
        );
      }
    }
  }

  Future<void> _save() async {
    if (_saving || _settings == null || !widget.feed.canManage) return;
    final error = providerSelectionError(
      textModels: _text,
      speechModels: _speech,
      textModel: _textId,
      speechModel: _speechId,
      voice: _voice,
      hasKey: _settings!.hasKey,
      replacementKey: _key.text,
    );
    if (error != null) {
      setState(() => _error = error);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.api.saveSettings(widget.feed.id, {
        'revision': _settings!.revision,
        'textModel': _textId,
        'speechModel': _speechId,
        'voice': _voice,
        if (_key.text.isNotEmpty) 'openRouterKey': _key.text,
      });
      if (mounted) {
        _key.clear();
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Saved for future briefings.')),
        );
        await _load();
      }
    } catch (e) {
      if (mounted) {
        setState(
          () =>
              _error = e is ApiFailure ? e.message : 'Could not save settings.',
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  void dispose() {
    _key.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final selected = _speech.where((m) => m.id == _speechId).firstOrNull;
    return Scaffold(
      appBar: AppBar(title: Text(widget.feed.name)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(
            'Make it sound like you.',
            style: Theme.of(context).textTheme.headlineMedium,
          ),
          const SizedBox(height: 12),
          Text(
            widget.feed.kind == 'team'
                ? 'Shared choices for everyone in this space. Changes apply to future briefings.'
                : 'Choose how your next briefings sound.',
          ),
          const SizedBox(height: 28),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 16),
              child: Text(
                _error!,
                semanticsLabel: 'Error: $_error',
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ),
          if (_settings == null) ...[
            if (_error == null)
              const Center(child: CircularProgressIndicator())
            else
              FilledButton(onPressed: _load, child: const Text('Try again')),
          ] else ...[
            if (!widget.feed.canManage)
              const Padding(
                padding: EdgeInsets.only(bottom: 20),
                child: Text(
                  'A space administrator manages models, voices and the shared key.',
                ),
              ),
            DropdownButtonFormField<String>(
              value: _text.any((m) => m.id == _textId) ? _textId : null,
              isExpanded: true,
              decoration: const InputDecoration(labelText: 'Briefing model'),
              items: _text
                  .map(
                    (m) => DropdownMenuItem(
                      value: m.id,
                      child: Text(m.name, overflow: TextOverflow.ellipsis),
                    ),
                  )
                  .toList(),
              onChanged: widget.feed.canManage && !_saving
                  ? (v) => setState(() => _textId = v)
                  : null,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: selected?.id,
              isExpanded: true,
              decoration: const InputDecoration(labelText: 'Speech model'),
              items: _speech
                  .map(
                    (m) => DropdownMenuItem(
                      value: m.id,
                      child: Text(m.name, overflow: TextOverflow.ellipsis),
                    ),
                  )
                  .toList(),
              onChanged: widget.feed.canManage && !_saving
                  ? (v) => setState(() {
                      _speechId = v;
                      _voice = null;
                    })
                  : null,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: selected?.voices.contains(_voice) == true ? _voice : null,
              isExpanded: true,
              decoration: const InputDecoration(labelText: 'Voice'),
              items: (selected?.voices ?? <String>[])
                  .map((v) => DropdownMenuItem(value: v, child: Text(v)))
                  .toList(),
              onChanged: widget.feed.canManage && !_saving
                  ? (v) => setState(() => _voice = v)
                  : null,
            ),
            if (widget.feed.canManage) ...[
              const SizedBox(height: 24),
              TextField(
                controller: _key,
                enabled: !_saving,
                obscureText: true,
                autocorrect: false,
                enableSuggestions: false,
                decoration: InputDecoration(
                  labelText: _settings!.hasKey
                      ? 'Replace OpenRouter key'
                      : 'OpenRouter key',
                  helperText: _settings!.hasKey
                      ? 'A key is connected. Leave blank to keep it.'
                      : 'Your key pays for narration. It is never shared with agents.',
                ),
              ),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _saving ? null : _save,
                child: Text(_saving ? 'Saving…' : 'Save choices'),
              ),
            ],
          ],
        ],
      ),
    );
  }
}
