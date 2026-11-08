import 'native_session.dart';

/// Explicit release inputs, not proof of server registration or current access.
class ServiceConfiguration {
  final Uri origin;
  final NativeSessionConfiguration session;

  ServiceConfiguration._(this.origin, this.session);

  /// Returns disabled only when all inputs are absent; partial inputs refuse.
  static ServiceConfiguration? parse({
    required String origin,
    required String issuer,
    required String audience,
  }) {
    if (origin.isEmpty && issuer.isEmpty && audience.isEmpty) return null;
    try {
      final api = _origin(origin);
      final authority = _origin(issuer);
      final resource = Uri.parse(audience);
      if (!_ascii(audience) ||
          audience.length > 2048 ||
          resource.scheme != 'https' ||
          resource.host.isEmpty ||
          resource.userInfo.isNotEmpty ||
          resource.hasQuery ||
          resource.hasFragment ||
          resource.path.isEmpty ||
          audience != resource.toString() ||
          audience.contains('%') ||
          resource.path.contains('//') ||
          resource.pathSegments.any((s) => s == '.' || s == '..') ||
          !audience.startsWith('${_origin(resource.origin)}/')) {
        throw const FormatException();
      }
      return ServiceConfiguration._(
        api,
        NativeSessionConfiguration(issuer: authority, audience: audience),
      );
    } on Object {
      throw const FormatException('Invalid service configuration');
    }
  }

  /// Constructs the real native client without network or platform effects.
  NativeSession createSession() => NativeSession(session);

  static Uri _origin(String value) {
    final uri = Uri.parse(value);
    if (!_ascii(value) ||
        value.length > 2048 ||
        uri.scheme != 'https' ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        uri.path.isNotEmpty ||
        uri.origin != value) {
      throw const FormatException();
    }
    return uri;
  }

  static bool _ascii(String value) =>
      value.codeUnits.every((c) => c >= 0x21 && c <= 0x7e);
}
