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
  bool _saving = false, _notificationSaving = false, _catalogLoading = false;
  bool? _notifications;
  String? _notificationError, _notificationMessage, _catalogError;
  int _revision = 0;

  bool get _busy => _saving || _notificationSaving;
  bool _current(int revision) => mounted && revision == _revision;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  @override
  void didUpdateWidget(covariant SettingsScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!identical(oldWidget.api, widget.api) ||
        oldWidget.feed.id != widget.feed.id ||
        oldWidget.feed.canManage != widget.feed.canManage ||
        oldWidget.feed.kind != widget.feed.kind) {
      _key.clear();
      _saving = false;
      unawaited(_load());
    }
  }

  Future<void> _load() async {
    final revision = ++_revision;
    final api = widget.api, feed = widget.feed.id;
    setState(() {
      _settings = null;
      _notifications = null;
      _notificationSaving = false;
      _notificationError = _notificationMessage = _catalogError = _error = null;
      _catalogLoading = true;
      _text = [];
      _speech = [];
      _textId = _speechId = _voice = null;
    });
    try {
      final settings = await api.settings(feed);
      if (!_current(revision)) return;
      setState(() {
        _settings = settings;
        _notifications = settings.notifications;
        _textId = settings.textModel;
        _speechId = settings.speechModel;
        _voice = settings.voice;
      });
    } catch (e) {
      if (!_current(revision)) return;
      setState(() {
        _catalogLoading = false;
        _error = e is ApiFailure ? e.message : 'Settings could not be loaded.';
        _notificationError =
            'Your notification preference could not be loaded.';
      });
      return;
    }
    // Provider catalogs do not control a person's notification preference.
    try {
      final catalogs = await Future.wait([
        api.models('text'),
        api.models('speech'),
      ]);
      if (!_current(revision)) return;
      setState(() {
        _text = catalogs[0];
        _speech = catalogs[1];
      });
    } catch (error) {
      if (!_current(revision)) return;
      if (error is ApiFailure && (error.status == 401 || error.status == 403)) {
        _key.clear();
      }
      setState(() => _catalogError = 'Model choices could not be loaded.');
    } finally {
      if (_current(revision)) setState(() => _catalogLoading = false);
    }
  }

  String _preferenceFailure(Object error) => switch (error) {
    ApiFailure(status: 401) => 'Your session has expired. Sign in again.',
    ApiFailure(status: 403) => 'You no longer have access to this space.',
    _ =>
      'Could not confirm your preference. Refresh to check before changing it.',
  };

  void _preferenceFailed(Object error) {
    _notifications = null;
    _notificationError = _preferenceFailure(error);
    _notificationMessage = null;
    if (error is ApiFailure && (error.status == 401 || error.status == 403)) {
      _key.clear();
      _settings = null;
      _error = _notificationError;
    }
  }

  Future<void> _refreshPreference() async {
    if (_busy) return;
    if (_settings == null) {
      await _load();
      return;
    }
    final revision = _revision;
    final api = widget.api, feed = widget.feed.id;
    setState(() {
      _notificationSaving = true;
      _notificationError = _notificationMessage = null;
    });
    try {
      final current = await api.settings(feed);
      if (!_current(revision)) return;
      setState(() {
        _notifications = current.notifications;
        _notificationMessage = 'Current preference confirmed.';
      });
    } catch (e) {
      if (_current(revision)) setState(() => _preferenceFailed(e));
    } finally {
      if (_current(revision)) setState(() => _notificationSaving = false);
    }
  }

  Future<void> _setNotifications(bool enabled) async {
    if (_busy || _notifications == null || enabled == _notifications) return;
    final revision = _revision;
    final api = widget.api, feed = widget.feed.id;
    setState(() {
      _notificationSaving = true;
      _notificationError = _notificationMessage = null;
    });
    try {
      Object? writeError;
      try {
        await api.notifications(feed, enabled);
      } catch (e) {
        writeError = e;
      }
      if (!_current(revision)) return;
      if (writeError is ApiFailure &&
          (writeError.status == 401 || writeError.status == 403)) {
        throw writeError;
      }
      // A response may be lost after commit. Read once; never repeat the write.
      final current = await api.settings(feed);
      if (!_current(revision)) return;
      setState(() {
        _notifications = current.notifications;
        _notificationMessage =
            writeError == null && current.notifications == enabled
            ? 'Saved for you in this feed.'
            : 'Current preference confirmed.';
      });
    } catch (e) {
      if (_current(revision)) setState(() => _preferenceFailed(e));
    } finally {
      if (_current(revision)) setState(() => _notificationSaving = false);
    }
  }

  Future<void> _save() async {
    if (_busy ||
        _settings == null ||
        !widget.feed.canManage ||
        _catalogLoading ||
        _catalogError != null) {
      return;
    }
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
    final revision = _revision;
    final api = widget.api, feed = widget.feed.id;
    final body = {
      'revision': _settings!.revision,
      'textModel': _textId,
      'speechModel': _speechId,
      'voice': _voice,
      if (_key.text.isNotEmpty) 'openRouterKey': _key.text,
    };
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await api.saveSettings(feed, body);
      if (!mounted || revision != _revision) return;
      _key.clear();
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Saved for future briefings.')),
      );
      await _load();
    } catch (e) {
      if (_current(revision)) {
        if (e is ApiFailure && (e.status == 401 || e.status == 403)) {
          _key.clear();
        }
        setState(
          () =>
              _error = e is ApiFailure ? e.message : 'Could not save settings.',
        );
      }
    } finally {
      if (_current(revision)) setState(() => _saving = false);
    }
  }

  @override
  void dispose() {
    ++_revision;
    _key.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final revision = _revision;
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
                ? 'Models and voices are shared in this space. Changes apply to future briefings.'
                : 'Choose how your next briefings sound.',
          ),
          const SizedBox(height: 28),
          SwitchListTile.adaptive(
            contentPadding: EdgeInsets.zero,
            title: const Text('Arrival notifications'),
            value: _notifications ?? false,
            onChanged: _notifications == null || _busy
                ? null
                : (enabled) {
                    if (_current(revision)) {
                      unawaited(_setNotifications(enabled));
                    }
                  },
            subtitle: Text(
              widget.api.notificationPreferencesAreLocal
                  ? 'Local preview only. This preference resets with the preview; device notifications are not sent.'
                  : 'Only for you in this feed. Delivery also requires notification permission on your device.',
            ),
          ),
          if (_notificationSaving)
            const Text('Checking preference…')
          else if (_notificationError != null)
            Text(
              _notificationError!,
              semanticsLabel: 'Error: $_notificationError',
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            )
          else if (_notifications == null)
            const Text('Loading your preference…')
          else if (_notificationMessage != null)
            Text(_notificationMessage!),
          if (_notificationError != null)
            Align(
              alignment: Alignment.centerLeft,
              child: TextButton(
                onPressed: _busy
                    ? null
                    : () {
                        if (_current(revision)) unawaited(_refreshPreference());
                      },
                child: const Text('Refresh preference'),
              ),
            ),
          const SizedBox(height: 28),
          if (_catalogError != null)
            Text(
              _catalogError!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
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
              FilledButton(
                onPressed: () {
                  if (_current(revision)) unawaited(_load());
                },
                child: const Text('Try again'),
              ),
          ] else ...[
            if (!widget.feed.canManage)
              const Padding(
                padding: EdgeInsets.only(bottom: 20),
                child: Text(
                  'A space administrator manages models, voices and the shared key.',
                ),
              ),
            DropdownButtonFormField<String>(
              initialValue: _text.any((m) => m.id == _textId) ? _textId : null,
              isExpanded: true,
              key: ValueKey('text-$_revision-$_textId'),
              decoration: const InputDecoration(labelText: 'Briefing model'),
              items: _text
                  .map(
                    (m) => DropdownMenuItem(
                      value: m.id,
                      child: Text(m.name, overflow: TextOverflow.ellipsis),
                    ),
                  )
                  .toList(),
              onChanged:
                  widget.feed.canManage &&
                      !_busy &&
                      !_catalogLoading &&
                      _catalogError == null
                  ? (v) {
                      if (_current(revision)) setState(() => _textId = v);
                    }
                  : null,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              initialValue: selected?.id,
              isExpanded: true,
              key: ValueKey('speech-$_revision-$_speechId'),
              decoration: const InputDecoration(labelText: 'Speech model'),
              items: _speech
                  .map(
                    (m) => DropdownMenuItem(
                      value: m.id,
                      child: Text(m.name, overflow: TextOverflow.ellipsis),
                    ),
                  )
                  .toList(),
              onChanged:
                  widget.feed.canManage &&
                      !_busy &&
                      !_catalogLoading &&
                      _catalogError == null
                  ? (v) {
                      if (_current(revision)) {
                        setState(() {
                          _speechId = v;
                          _voice = null;
                        });
                      }
                    }
                  : null,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              initialValue: selected?.voices.contains(_voice) == true
                  ? _voice
                  : null,
              isExpanded: true,
              key: ValueKey('voice-$_revision-$_voice'),
              decoration: const InputDecoration(labelText: 'Voice'),
              items: (selected?.voices ?? <String>[])
                  .map((v) => DropdownMenuItem(value: v, child: Text(v)))
                  .toList(),
              onChanged:
                  widget.feed.canManage &&
                      !_busy &&
                      !_catalogLoading &&
                      _catalogError == null
                  ? (v) {
                      if (_current(revision)) setState(() => _voice = v);
                    }
                  : null,
            ),
            if (widget.feed.canManage) ...[
              const SizedBox(height: 24),
              TextField(
                key: ValueKey('key-$_revision'),
                controller: _key,
                enabled: !_busy,
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
                onPressed: _busy || _catalogLoading || _catalogError != null
                    ? null
                    : () {
                        if (_current(revision)) unawaited(_save());
                      },
                child: Text(_saving ? 'Saving…' : 'Save choices'),
              ),
            ],
          ],
        ],
      ),
    );
  }
}
