import Foundation
import UIKit
import Darwin

// Main actor isolation serializes foreground samples and CPU baseline resets.
@MainActor final class StockMetrics {
    private let clock = ContinuousClock()
    private let origin = ContinuousClock.now
    private let dateFormatter = ISO8601DateFormatter()
    private var cpu = CPUSampler()

    nonisolated static func modelIdentifier() -> String? {
        var info = utsname()
        guard uname(&info) == 0 else { return nil }
        let capacity = MemoryLayout.size(ofValue: info.machine)
        return withUnsafePointer(to: &info.machine) {
            $0.withMemoryRebound(to: CChar.self, capacity: capacity) {
                String(cString: $0)
            }
        }
    }

    func resetCPU() { cpu.reset() }

    func sample(directory: URL?) -> [String: Any] {
        let device = UIDevice.current
        device.isBatteryMonitoringEnabled = true
        let level = batteryPercentage(level: device.batteryLevel)
        let state: Result<String, ProbeFailure>
        switch device.batteryState {
        case .charging: state = .success("charging")
        case .full: state = .success("full")
        case .unplugged: state = .success("unplugged")
        default: state = .failure(.unknownReading)
        }

        let process = ProcessInfo.processInfo
        let physicalMemory = process.physicalMemory
        let activeProcessors = process.activeProcessorCount
        let timestamp = dateFormatter.string(from: Date())
        let sample = cpuTicks(activeProcessors: activeProcessors)
        let usage = cpu.update(sample, ticksPerSecond: sysconf(_SC_CLK_TCK))
        let memory = memoryUsage(physicalMemory: physicalMemory)
        let (volume, available) = storage(directory: directory)

        func reading<T>(_ result: Result<T, ProbeFailure>, _ scope: String, _ unit: String) -> [String: Any] {
            Reading(result: result, scope: scope, unit: unit, sampledAt: timestamp).payload
        }

        let probes: [String: [String: Any]] = [
            "cpuTicks": reading(sample.map(\.ticks), "system logical processors", "Mach ticks [user, system, nice, idle] per processor"),
            "cpuPercent": reading(usage.map(\.percent), "system logical processors", "percent"),
            "cpuSampleInterval": reading(usage.map(\.interval), "system CPU sample", "seconds"),
            "logicalProcessors": reading(process.processorCount > 0 ? .success(process.processorCount) : .failure(.invalidReading),
                                         "system", "logical processors"),
            "activeLogicalProcessors": reading(activeProcessors > 0 ? .success(activeProcessors) : .failure(.invalidReading),
                                               "system", "logical processors"),
            "physicalMemory": reading(physicalMemory > 0 ? .success(physicalMemory) : .failure(.invalidReading), "device", "bytes"),
            "freeMemory": reading(memory.map(\.free), "host VM free pages excluding speculative pages", "bytes"),
            "usedMemory": reading(memory.map(\.darwinUsed), "host VM physical memory less Mach free and inactive pages", "bytes"),
            "availableMemory": reading(memory.map(\.darwinAvailable), "host VM Mach free (including speculative) and inactive pages", "bytes"),
            "batteryPercent": reading(level, "device battery", "percent"),
            "batteryState": reading(state, "device battery", "state"),
            "volumeCapacity": reading(volume, "volume containing app directory", "bytes"),
            "availableToApp": reading(available, "app directory capacity for important usage", "bytes"),
            "model": reading(Self.modelIdentifier().map(Result<String, ProbeFailure>.success) ?? .failure(.unknownReading), "device", "model identifier"),
            "name": reading(.success(device.name), "UIKit device identity", "name"),
            "osVersion": reading(.success(device.systemVersion), "device OS", "version")
        ]
        // Preserve the original local diagnostic keys alongside per-reading metadata.
        var snapshot: [String: Any] = [
            "batteryState": (try? state.get()) ?? "unknown",
            "sampledAt": timestamp,
            "probes": probes
        ]
        if physicalMemory > 0 { snapshot["physicalMemory"] = physicalMemory }
        snapshot["batteryPercent"] = try? level.get()
        snapshot["volumeCapacity"] = try? volume.get()
        snapshot["availableToApp"] = try? available.get()
        return snapshot
    }

    private func cpuTicks(activeProcessors: Int) -> Result<CPUTickSample, ProbeFailure> {
        let capacity: UInt32 = 64 * 4
        var ticks = [UInt32](repeating: 0, count: Int(capacity))
        var cores: UInt32 = 0
        let status = BeszelHostCPUTicks(&ticks, capacity, &cores)
        guard status == KERN_SUCCESS else { return .failure(Self.machFailure(status)) }
        let elapsed = origin.duration(to: clock.now).components
        let seconds = Double(elapsed.seconds) + Double(elapsed.attoseconds) * 1e-18
        let values = stride(from: 0, to: Int(cores) * 4, by: 4).map { Array(ticks[$0..<($0 + 4)]) }
        return .success(CPUTickSample(ticks: values, time: seconds, activeProcessors: activeProcessors))
    }

    private func memoryUsage(physicalMemory: UInt64) -> Result<HostMemory, ProbeFailure> {
        var freePages: UInt64 = 0
        var speculativePages: UInt64 = 0
        var inactivePages: UInt64 = 0
        var pageSize: UInt64 = 0
        let status = BeszelHostVMPages(&freePages, &speculativePages, &inactivePages, &pageSize)
        guard status == KERN_SUCCESS else { return .failure(Self.machFailure(status)) }
        return hostMemory(freePages: freePages, speculativePages: speculativePages, inactivePages: inactivePages,
                          pageSize: pageSize, physicalMemory: physicalMemory)
    }

    private func storage(directory: URL?) -> (Result<UInt64, ProbeFailure>, Result<UInt64, ProbeFailure>) {
        guard let directory else { return (.failure(.missingDirectory), .failure(.missingDirectory)) }
        do {
            let values = try directory.resourceValues(forKeys: [.volumeTotalCapacityKey, .volumeAvailableCapacityForImportantUsageKey])
            let total = storageCapacity(values.volumeTotalCapacity)
            let available = storageCapacity(values.volumeAvailableCapacityForImportantUsage)
            if let size = try? total.get(), let free = try? available.get(), free > size {
                return (total, .failure(.invalidReading))
            }
            return (total, available)
        } catch let error as NSError {
            let failure = ProbeFailure.resourceError(domain: error.domain, code: error.code)
            return (.failure(failure), .failure(failure))
        }
    }

    private static func machFailure(_ status: Int32) -> ProbeFailure {
        switch status {
        case KERN_NO_ACCESS, KERN_PROTECTION_FAILURE: return .accessDenied(status)
        case KERN_NOT_SUPPORTED, MIG_BAD_ID: return .unsupported(status)
        case KERN_INVALID_VALUE: return .invalidReading
        default: return .apiError(status)
        }
    }
}
