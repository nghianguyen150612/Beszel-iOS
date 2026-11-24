import Foundation

enum ProbeFailure: Error, Equatable {
    case accessDenied(Int32)
    case unsupported(Int32)
    case apiError(Int32)
    case invalidReading
    case baselineRequired
    case discontinuity
    case unknownReading
    case missingCapacity
    case missingDirectory
    case resourceError(domain: String, code: Int)

    var description: String {
        switch self {
        case .accessDenied(let code): return "access denied (Mach \(code))"
        case .unsupported(let code): return "unsupported (Mach \(code))"
        case .apiError(let code): return "Mach error \(code)"
        case .invalidReading: return "invalid reading"
        case .baselineRequired: return "CPU baseline required"
        case .discontinuity: return "CPU interval or processor count changed"
        case .unknownReading: return "OS reported an unknown value"
        case .missingCapacity: return "OS did not report capacity"
        case .missingDirectory: return "app directory unavailable"
        case .resourceError(let domain, let code): return "volume query failed (\(domain) \(code))"
        }
    }
}

struct Reading<Value> {
    let result: Result<Value, ProbeFailure>
    let scope: String
    let unit: String
    let sampledAt: String

    var payload: [String: Any] {
        var resultPayload: [String: Any] = ["scope": scope, "unit": unit, "sampledAt": sampledAt]
        switch result {
        case .success(let value):
            resultPayload["available"] = true
            resultPayload["value"] = value
        case .failure(let failure):
            resultPayload["available"] = false
            resultPayload["reason"] = failure.description
        }
        return resultPayload
    }
}

struct CPUTickSample {
    // Mach reports user, system, nice and idle as cumulative 32-bit counters.
    let ticks: [[UInt32]]
    let time: Double
    let activeProcessors: Int
}

struct CPUUsage: Equatable {
    let percent: Double
    let interval: Double
}

struct CPUSampler {
    static let cadence: TimeInterval = 2
    private var previous: CPUTickSample?

    mutating func reset() { previous = nil }

    mutating func update(_ reading: Result<CPUTickSample, ProbeFailure>, ticksPerSecond: Int) -> Result<CPUUsage, ProbeFailure> {
        switch reading {
        case .failure(let reason):
            reset()
            return .failure(reason)
        case .success(let sample):
            guard sample.time.isFinite, !sample.ticks.isEmpty, sample.ticks.allSatisfy({ $0.count == 4 }) else {
                reset()
                return .failure(.invalidReading)
            }
            defer { previous = sample }
            guard let previous else { return .failure(.baselineRequired) }
            let interval = sample.time - previous.time
            guard interval >= 0.5, interval <= 3 * Self.cadence, sample.ticks.count == previous.ticks.count,
                  sample.activeProcessors == previous.activeProcessors else {
                return .failure(.discontinuity)
            }
            // Allow tick quantization and time spent traversing the processors.
            // A backward jump only counts as a wrap if its modular delta fits.
            guard ticksPerSecond > 0 else { return .failure(.invalidReading) }
            let budget = 2 * interval * Double(ticksPerSecond) + 8
            var busy: UInt64 = 0
            var total: UInt64 = 0
            for (current, old) in zip(sample.ticks, previous.ticks) {
                let deltas = zip(current, old).map { UInt64($0.0 &- $0.1) }
                let coreTotal = deltas.reduce(0, +)
                guard Double(coreTotal) <= budget else { return .failure(.invalidReading) }
                busy += coreTotal - deltas[3]
                total += coreTotal
            }
            guard total > 0 else { return .failure(.invalidReading) }
            return .success(CPUUsage(percent: 100 * Double(busy) / Double(total), interval: interval))
        }
    }
}

struct HostMemory: Equatable {
    let free: UInt64
    let darwinAvailable: UInt64
    let darwinUsed: UInt64
}

func hostMemory(freePages: UInt64, speculativePages: UInt64, inactivePages: UInt64,
                pageSize: UInt64, physicalMemory: UInt64) -> Result<HostMemory, ProbeFailure> {
    guard pageSize > 0, physicalMemory > 0, physicalMemory % pageSize == 0, speculativePages <= freePages else {
        return .failure(.invalidReading)
    }
    let pages = freePages.addingReportingOverflow(inactivePages)
    let available = pages.partialValue.multipliedReportingOverflow(by: pageSize)
    guard !pages.overflow, !available.overflow, available.partialValue <= physicalMemory else {
        return .failure(.invalidReading)
    }
    // Mach's free_count includes speculative pages. Beszel's default Darwin
    // backend counts free_count plus inactive_count as available.
    return .success(HostMemory(free: (freePages - speculativePages) * pageSize,
                               darwinAvailable: available.partialValue,
                               darwinUsed: physicalMemory - available.partialValue))
}

func batteryPercentage(level: Float) -> Result<Double, ProbeFailure> {
    guard (0...1).contains(level) else { return .failure(.unknownReading) }
    return .success(Double(level) * 100)
}

func storageCapacity<T: BinaryInteger>(_ bytes: T?) -> Result<UInt64, ProbeFailure> {
    guard let bytes else { return .failure(.missingCapacity) }
    guard let value = UInt64(exactly: bytes) else { return .failure(.invalidReading) }
    return .success(value)
}
