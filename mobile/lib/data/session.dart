import 'api.dart';

// NativeSession is an explicitly configured candidate for the AMOS personal
// authorization-code/PKCE seam. Production composition stays disabled until the
// exact issuer, audience, public client and iOS redirect are registered and
// qualified. MCP credentials are not human sessions; no refresh/OIDC is assumed.
abstract interface class InteractiveSession implements AccessSession {
  Future<bool> restore();
  Future<void> signIn();
}
