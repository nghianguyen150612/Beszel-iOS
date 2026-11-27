import Foundation
import Security

enum TokenStore {
    private static var query: [String: Any] { [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: "BeszelStockAgent",
        kSecAttrAccount as String: "registration-token",
        kSecAttrSynchronizable as String: false
    ] }

    typealias Copy = (CFDictionary, UnsafeMutablePointer<CFTypeRef?>?) -> OSStatus
    typealias Update = (CFDictionary, CFDictionary) -> OSStatus
    typealias Add = (CFDictionary, UnsafeMutablePointer<CFTypeRef?>?) -> OSStatus

    static func load(copy: Copy = SecItemCopyMatching) throws -> String {
        var q = query
        q[kSecReturnData as String] = true
        var value: CFTypeRef?
        let result = copy(q as CFDictionary, &value)
        if result == errSecItemNotFound { return "" }
        guard result == errSecSuccess else { throw NSError(domain: NSOSStatusErrorDomain, code: Int(result)) }
        guard let data = value as? Data, let token = String(data: data, encoding: .utf8) else {
            throw NSError(domain: NSOSStatusErrorDomain, code: Int(errSecDecode))
        }
        return token
    }
    static func save(_ token: String, update: Update = SecItemUpdate, add: Add = SecItemAdd) throws {
        let attributes: [String: Any] = [kSecValueData as String: Data(token.utf8), kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly]
        let result = update(query as CFDictionary, attributes as CFDictionary)
        if result == errSecItemNotFound {
            let status = add(query.merging(attributes) { _, new in new } as CFDictionary, nil)
            guard status == errSecSuccess else { throw NSError(domain: NSOSStatusErrorDomain, code: Int(status)) }
        } else if result != errSecSuccess { throw NSError(domain: NSOSStatusErrorDomain, code: Int(result)) }
    }
}
