import Foundation
import Security

@main struct TokenStoreTests {
    static func main() throws {
        func verifyQuery(_ query: CFDictionary) {
            let attributes = query as NSDictionary
            precondition(attributes[kSecClass] as? String == kSecClassGenericPassword as String)
            precondition(attributes[kSecAttrSynchronizable] as? Bool == false)
        }

        let missing = try TokenStore.load { query, _ in
            verifyQuery(query)
            return errSecItemNotFound
        }
        precondition(missing.isEmpty)
        let loaded = try TokenStore.load { query, result in
            verifyQuery(query)
            precondition((query as NSDictionary)[kSecReturnData] as? Bool == true)
            result?.pointee = Data("disposable".utf8) as CFData
            return errSecSuccess
        }
        precondition(loaded == "disposable")

        func verifyToken(_ attributes: CFDictionary) {
            let values = attributes as NSDictionary
            precondition(values[kSecValueData] as? Data == Data("disposable".utf8))
            precondition(values[kSecAttrAccessible] as? String == kSecAttrAccessibleWhenUnlockedThisDeviceOnly as String)
        }

        try TokenStore.save("disposable", update: { query, attrs in
            verifyQuery(query)
            verifyToken(attrs)
            return errSecSuccess
        }, add: { _, _ in fatalError("updated item must not be added again") })

        var inserted = false
        try TokenStore.save("disposable", update: { _, _ in errSecItemNotFound }, add: { attrs, _ in
            verifyQuery(attrs)
            verifyToken(attrs)
            inserted = true
            return errSecSuccess
        })
        precondition(inserted)

        do {
            _ = try TokenStore.load { _, _ in errSecInteractionNotAllowed }
            fatalError("Keychain access failure was ignored")
        } catch let error as NSError {
            precondition(error.code == Int(errSecInteractionNotAllowed))
        }
        do {
            try TokenStore.save("disposable", update: { _, _ in errSecAuthFailed },
                                add: { _, _ in fatalError("add after a denied update") })
            fatalError("Keychain authentication failure was ignored")
        } catch let error as NSError {
            precondition(error.code == Int(errSecAuthFailed))
        }
        print("PASS: Keychain query, replacement, insertion and failure handling (injected Security calls)")
    }
}
