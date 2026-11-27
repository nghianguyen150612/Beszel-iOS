import Foundation
import SwiftUI
import UIKit

func goString(_ pointer: UnsafeMutablePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    defer { BeszelGoFree(pointer) }
    return String(cString: pointer)
}

@MainActor final class AgentController: ObservableObject {
    @Published var hubURL = UserDefaults.standard.string(forKey: "hubURL") ?? ""
    @Published var hubKey = UserDefaults.standard.string(forKey: "hubKey") ?? ""
    @Published var token = ""
    @Published var state = "disconnected"
    @Published var diagnostics = ""
    @Published var error = ""
    private let worker = DispatchQueue(label: "beszel.agent.runtime")
    private let metrics = StockMetrics()
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
        timer = Timer.scheduledTimer(withTimeInterval: CPUSampler.cadence, repeats: true) { [weak self] _ in
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
        let config: [String: Any] = ["hubURL": hubURL, "hubKey": hubKey, "token": token, "dataDirectory": directory.path, "model": StockMetrics.modelIdentifier() ?? "unavailable", "osVersion": UIDevice.current.systemVersion, "name": UIDevice.current.name]
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
        if !active {
            metrics.resetCPU()
            disconnect()
        } else {
            refresh()
        }
        // Resume requires an explicit Connect. No indefinite background claim.
    }
    func refresh() {
        guard active else { return }
        let sample = metrics.sample(directory: directory)
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
