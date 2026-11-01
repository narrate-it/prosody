import 'api.dart';

// Implement only against a reviewed, qualified AMOS human mobile session seam.
// The current native issuer has no refresh token/OIDC ID token, accepts only
// registered numeric loopback redirects, and does not qualify iOS browser return
// or general bearer consumption APIs. MCP credentials are not human sessions.
// There is deliberately no production implementation or fallback credential.
abstract interface class InteractiveSession implements AccessSession {
  Future<bool> restore();
  Future<void> signIn();
}
