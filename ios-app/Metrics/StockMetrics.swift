import Foundation
import UIKit

// Public APIs only. No process value is reported as whole-system used memory.
enum StockMetrics {
    static func modelIdentifier() -> String {
        var info = utsname()
        guard uname(&info) == 0 else { return "unavailable" }
        let capacity = MemoryLayout.size(ofValue: info.machine)
        return withUnsafePointer(to: &info.machine) {
            $0.withMemoryRebound(to: CChar.self, capacity: capacity) {
                String(cString: $0)
            }
        }
    }

    @MainActor static func sample(directory: URL) -> [String: Any] {
        let device = UIDevice.current
        device.isBatteryMonitoringEnabled = true
        let level = device.batteryLevel
        let state: String
        switch device.batteryState {
        case .charging: state = "charging"
        case .full: state = "full"
        case .unplugged: state = "unplugged"
        default: state = "unknown"
        }
        var result: [String: Any] = [
            "physicalMemory": ProcessInfo.processInfo.physicalMemory,
            "batteryState": state,
            "sampledAt": ISO8601DateFormatter().string(from: Date())
        ]
        if level >= 0 && level <= 1 { result["batteryPercent"] = Double(level) * 100 }
        if let values = try? directory.resourceValues(forKeys: [.volumeTotalCapacityKey, .volumeAvailableCapacityForImportantUsageKey]) {
            if let total = values.volumeTotalCapacity, total >= 0 { result["volumeCapacity"] = UInt64(total) }
            if let available = values.volumeAvailableCapacityForImportantUsage, available >= 0 { result["availableToApp"] = UInt64(available) }
        }
        return result
    }
}
