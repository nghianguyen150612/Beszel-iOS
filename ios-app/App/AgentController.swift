import Foundation
import SwiftUI
import Security
import UIKit

func goString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    defer { BeszelGoFree(pointer) }
    return String(cString: pointer)
}

enum TokenStore {
    private static let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: "BeszelStockAgent",
        kSecAttrAccount as String: "registration-token",
        kSecAttrSynchronizable as String: false
    ]
    static func load() throws -> String {
        var q = query
        q[kSecReturnData as String] = true
        var value: CFTypeRef?
        let result = SecItemCopyMatching(q as CFDictionary, &value)
        if result == errSecItemNotFound { return "" }
        guard result == errSecSuccess, let data = value as? Data, let token = String(data: data, encoding: .utf8) else {
            throw NSError(domain: NSOSStatusErrorDomain, code: Int(result))
        }
        return token
    }
    static func save(_ token: String) throws {
        let attributes: [String: Any] = [kSecValueData as String: Data(token.utf8), kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly]
        let result = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if result == errSecItemNotFound {
            let status = SecItemAdd(query.merging(attributes) { _, new in new } as CFDictionary, nil)
            guard status == errSecSuccess else { throw NSError(domain: NSOSStatusErrorDomain, code: Int(status)) }
        } else if result != errSecSuccess { throw NSError(domain: NSOSStatusErrorDomain, code: Int(result)) }
    }
}

@MainActor final class AgentController: ObservableObject {
    @Published var hubURL = UserDefaults.standard.string(forKey: "hubURL") ?? ""
    @Published var hubKey = UserDefaults.standard.string(forKey: "hubKey") ?? ""
    @Published var token = ""
    @Published var state = "disconnected"
    @Published var diagnostics = ""
    @Published var error = ""
    private let worker = DispatchQueue(label: "beszel.agent.runtime")
    private var timer: Timer?
    private var active = true
    private var directory: URL?

    init() {
        do {
            token = try TokenStore.load()
            let url = try FileManager.default.url(for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true).appendingPathComponent("Agent", isDirectory: true)
            try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
            var excluded = url
            var values = URLResourceValues(); values.isExcludedFromBackup = true
            try excluded.setResourceValues(values)
            directory = url
        } catch { self.error = "Configuration storage unavailable: \(error.localizedDescription)" }
        timer = Timer.scheduledTimer(withTimeInterval: 2, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.refresh() }
        }
    }
    func save() -> Bool {
        do {
            try TokenStore.save(token)
            UserDefaults.standard.set(hubURL, forKey: "hubURL")
            UserDefaults.standard.set(hubKey, forKey: "hubKey")
            error = ""
            return true
        } catch { self.error = "Keychain save failed: \(error.localizedDescription)"; return false }
    }
    func connect() {
        guard active, let directory, save() else { return }
        let config: [String: Any] = ["hubURL": hubURL, "hubKey": hubKey, "token": token, "dataDirectory": directory.path, "model": StockMetrics.modelIdentifier(), "osVersion": UIDevice.current.systemVersion, "name": UIDevice.current.name]
        guard let data = try? JSONSerialization.data(withJSONObject: config), let json = String(data: data, encoding: .utf8) else { return }
        state = "connecting"
        worker.async {
            BeszelAgentStop()
            let detail = json.withCString { goString(BeszelAgentStart($0)) }
            Task { @MainActor in self.error = detail }
        }
    }
    func disconnect() {
        worker.async { BeszelAgentStop() }
        state = active ? "disconnected" : "suspended/inactive"
    }
    func sceneChanged(_ phase: ScenePhase) {
        active = phase == .active
        if !active { disconnect() } else { refresh() }
        // Resume requires an explicit Connect. No indefinite background claim.
    }
    func refresh() {
        guard active, let directory else { return }
        let sample = StockMetrics.sample(directory: directory)
        guard let data = try? JSONSerialization.data(withJSONObject: sample), let json = String(data: data, encoding: .utf8) else { return }
        worker.async {
            let sampleError = json.withCString { goString(BeszelAgentUpdateMetrics($0)) }
            let status = goString(BeszelAgentStatusJSON())
            Task { @MainActor in
                guard self.active else { return }
                if let data = status.data(using: .utf8), let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                    self.state = object["state"] as? String ?? "disconnected"
                    self.diagnostics = status
                }
                if !sampleError.isEmpty { self.error = sampleError }
            }
        }
    }
}
