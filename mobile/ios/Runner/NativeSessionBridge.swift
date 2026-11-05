import AuthenticationServices
import CryptoKit
import Flutter
import Security
import UIKit

/// No access group, synchronisation, Keychain defaults or signing changes.
final class NativeSessionBridge: NSObject {
  // ASWebAuthenticationSession retains its provider weakly. Keep this immutable
  // anchor alive for the browser operation, even if the scene later resigns.
  private final class PresentationContext: NSObject, ASWebAuthenticationPresentationContextProviding {
    let window: UIWindow
    init(window: UIWindow) { self.window = window }
    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor { window }
  }
  private var channel: FlutterMethodChannel
  private var browser: ASWebAuthenticationSession?
  private var pending: FlutterResult?
  private var operation = UUID()
  private let windowProvider: () -> UIWindow?
  private var presentationContext: PresentationContext?

  // Resolve after the Flutter scene is connected, never at engine registration.
  static func foregroundWindow() -> UIWindow? {
    let windows = UIApplication.shared.connectedScenes
      .compactMap { $0 as? UIWindowScene }
      .filter { $0.activationState == .foregroundActive }
      .flatMap { $0.windows }
      .filter { $0.isKeyWindow && !$0.isHidden && $0.rootViewController != nil }
    // An ambiguous scene is not a safe presentation target.
    return windows.count == 1 ? windows.first : nil
  }

  init(messenger: FlutterBinaryMessenger, windowProvider: @escaping () -> UIWindow? = NativeSessionBridge.foregroundWindow) {
    channel = FlutterMethodChannel(name: "run.prosody/native_session", binaryMessenger: messenger)
    self.windowProvider = windowProvider
    super.init()
    channel.setMethodCallHandler { [weak self] call, result in
      guard let self else { result(FlutterError(code: "unavailable", message: "Session bridge unavailable", details: nil)); return }
      self.handle(call, result)
    }
  }
  private func fail(_ result: FlutterResult, _ code: String) {
    result(FlutterError(code: code, message: "Native session operation failed", details: nil))
  }
  private func encoded(_ bytes: Data) -> String {
    bytes.base64EncodedString().replacingOccurrences(of: "+", with: "-")
      .replacingOccurrences(of: "/", with: "_").replacingOccurrences(of: "=", with: "")
  }
  private func random() throws -> String {
    var bytes = [UInt8](repeating: 0, count: 32)
    guard SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes) == errSecSuccess else {
      throw NSError(domain: "NativeSession", code: 1)
    }
    return encoded(Data(bytes))
  }
  private func handle(_ call: FlutterMethodCall, _ result: @escaping FlutterResult) {
    let args = call.arguments as? [String: Any] ?? [:]
    switch call.method {
    case "proof":
      do {
        let verifier = try random()
        result(["verifier": verifier, "state": try random(), "challenge": encoded(Data(SHA256.hash(data: Data(verifier.utf8))))])
      } catch { fail(result, "random_failed") }
    case "cancel":
      operation = UUID()
      let previous = pending
      pending = nil
      browser?.cancel()
      browser = nil
      presentationContext = nil
      previous?(FlutterError(code: "cancelled", message: "Sign-in cancelled", details: nil))
      result(nil)
    case "authenticate":
      guard browser == nil, let text = args["url"] as? String, let url = URL(string: text),
            url.scheme == "https", url.host != nil else { fail(result, "invalid_request"); return }
      guard let window = windowProvider() else { fail(result, "presentation_unavailable"); return }
      let context = PresentationContext(window: window)
      presentationContext = context
      pending = result
      let identifier = UUID()
      operation = identifier
      let session = ASWebAuthenticationSession(url: url, callbackURLScheme: "run.prosody.prosody") { [weak self] url, error in
        DispatchQueue.main.async {
          guard let self, self.operation == identifier, let completion = self.pending else { return }
          self.pending = nil
          self.browser = nil
          self.presentationContext = nil
          if error != nil { self.fail(completion, "browser_cancelled_or_failed") }
          else if let url { completion(url.absoluteString) }
          else { self.fail(completion, "missing_callback") }
        }
      }
      browser = session
      session.presentationContextProvider = context
      session.prefersEphemeralWebBrowserSession = true
      if !session.start() { browser = nil; pending = nil; presentationContext = nil; fail(result, "browser_unavailable") }
    case "read", "write", "delete":
      guard let binding = args["binding"] as? String, !binding.isEmpty else { fail(result, "invalid_binding"); return }
      let account = encoded(Data(SHA256.hash(data: Data(binding.utf8))))
      let query: [String: Any] = [kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: "run.prosody.native-session.v1", kSecAttrAccount as String: account]
      if call.method == "delete" {
        let status = SecItemDelete(query as CFDictionary)
        if status == errSecSuccess || status == errSecItemNotFound { result(nil) } else { fail(result, "keychain_delete") }
      } else if call.method == "read" {
        var lookup = query
        lookup[kSecReturnData as String] = true
        lookup[kSecMatchLimit as String] = kSecMatchLimitOne
        var value: CFTypeRef?
        let status = SecItemCopyMatching(lookup as CFDictionary, &value)
        if status == errSecItemNotFound { result(nil); return }
        guard status == errSecSuccess, let data = value as? Data,
              let saved = try? JSONSerialization.jsonObject(with: data) as? [String: String] else { fail(result, "keychain_read"); return }
        result(saved)
      } else {
        guard let token = args["token"] as? String, let expires = args["expires"] as? String,
              let data = try? JSONSerialization.data(withJSONObject: ["token": token, "expires": expires]) else { fail(result, "invalid_grant"); return }
        let attributes: [String: Any] = [kSecValueData as String: data,
          kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly]
        var status = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
          var insert = query
          attributes.forEach { insert[$0.key] = $0.value }
          status = SecItemAdd(insert as CFDictionary, nil)
        }
        if status == errSecSuccess { result(nil) } else { fail(result, "keychain_write") }
      }
    default: result(FlutterMethodNotImplemented)
    }
  }
}
