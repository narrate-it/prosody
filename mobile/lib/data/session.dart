import 'api.dart';

// NativeSession is an explicitly configured candidate for the AMOS personal
// authorization-code/PKCE seam. Production composition stays disabled until the
// exact issuer, audience, public client and iOS redirect are registered and
// qualified. MCP credentials are not human sessions; no refresh/OIDC is assumed.
abstract interface class InteractiveSession implements AccessSession {
  /// Returns whether persisted human authority can be restored and validated.
  Future<bool> restore();
  /// Completes an interactive human sign-in or reports failure.
  Future<void> signIn();
}
