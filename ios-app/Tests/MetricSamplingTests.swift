import Foundation

@main struct MetricSamplingTests {
    static func main() {
        func ticks(_ time: Double, _ processors: [[UInt32]], active: Int = 1) -> Result<CPUTickSample, ProbeFailure> {
            .success(CPUTickSample(ticks: processors, time: time, activeProcessors: active))
        }

        var cpu = CPUSampler()
        precondition(cpu.update(ticks(0, [[10, 10, 0, 20]]), ticksPerSecond: 100) == .failure(.baselineRequired))
        precondition(cpu.update(ticks(2, [[70, 30, 0, 140]]), ticksPerSecond: 100) ==
                     .success(CPUUsage(percent: 40, interval: 2)))
        precondition(cpu.update(ticks(4, [[70, 30, 0, 340]]), ticksPerSecond: 100) ==
                     .success(CPUUsage(percent: 0, interval: 2)))
        precondition(cpu.update(ticks(6, [[269, 30, 1, 340]]), ticksPerSecond: 100) ==
                     .success(CPUUsage(percent: 100, interval: 2)))
        precondition(cpu.update(ticks(8, [[269, 30, 1, 340]]), ticksPerSecond: 100) == .failure(.invalidReading))
        precondition(cpu.update(ticks(8, [[270, 30, 1, 340]]), ticksPerSecond: 100) == .failure(.discontinuity))
        precondition(cpu.update(ticks(20, [[270, 31, 1, 341]]), ticksPerSecond: 100) == .failure(.discontinuity))
        precondition(cpu.update(ticks(22, [[270, 32, 1, 441]], active: 2), ticksPerSecond: 100) == .failure(.discontinuity))
        precondition(cpu.update(ticks(24, [[270, 33, 1, 541], [0, 0, 0, 0]], active: 2), ticksPerSecond: 100) == .failure(.discontinuity))
        precondition(cpu.update(ticks(26, [[1, 2]], active: 2), ticksPerSecond: 100) == .failure(.invalidReading))
        for failure in [ProbeFailure.accessDenied(8), .unsupported(46)] {
            precondition(cpu.update(.failure(failure), ticksPerSecond: 100) == .failure(failure))
            let probe = Reading(result: Result<Double, ProbeFailure>.failure(failure), scope: "system", unit: "percent", sampledAt: "test")
            precondition(probe.payload["available"] as? Bool == false)
            precondition(probe.payload["value"] == nil && probe.payload["reason"] as? String == failure.description)
            precondition(cpu.update(ticks(28, [[0, 0, 0, 0]]), ticksPerSecond: 100) == .failure(.baselineRequired))
        }

        cpu.reset()
        precondition(cpu.update(ticks(0, [[UInt32.max - 4, 0, 0, 0]]), ticksPerSecond: 100) == .failure(.baselineRequired))
        precondition(cpu.update(ticks(2, [[5, 0, 0, 10]]), ticksPerSecond: 100) ==
                     .success(CPUUsage(percent: 50, interval: 2)))
        precondition(cpu.update(ticks(4, [[4, 0, 0, 20]]), ticksPerSecond: 100) == .failure(.invalidReading))
        cpu.reset()
        precondition(cpu.update(ticks(6, [[100, 0, 0, 30]]), ticksPerSecond: 100) == .failure(.baselineRequired))

        let page: UInt64 = 16_384
        precondition(hostMemory(freePages: 10, speculativePages: 2, inactivePages: 20,
                                pageSize: page, physicalMemory: 100 * page) ==
                     .success(HostMemory(free: 8 * page, darwinAvailable: 30 * page, darwinUsed: 70 * page)))
        precondition(hostMemory(freePages: 1, speculativePages: 2, inactivePages: 0,
                                pageSize: page, physicalMemory: 100 * page) == .failure(.invalidReading))
        precondition(hostMemory(freePages: UInt64.max, speculativePages: 0, inactivePages: 1,
                                pageSize: page, physicalMemory: 100 * page) == .failure(.invalidReading))
        precondition(hostMemory(freePages: UInt64.max, speculativePages: 0, inactivePages: 0,
                                pageSize: page, physicalMemory: 100 * page) == .failure(.invalidReading))
        precondition(hostMemory(freePages: 10, speculativePages: 0, inactivePages: 100,
                                pageSize: page, physicalMemory: 100 * page) == .failure(.invalidReading))
        precondition(hostMemory(freePages: 10, speculativePages: 0, inactivePages: 0,
                                pageSize: 0, physicalMemory: 100 * page) == .failure(.invalidReading))

        precondition(batteryPercentage(level: 0) == .success(0))
        precondition(batteryPercentage(level: -1) == .failure(.unknownReading))
        precondition(batteryPercentage(level: .nan) == .failure(.unknownReading))
        precondition(storageCapacity(nil as Int64?) == .failure(.missingCapacity))
        precondition(storageCapacity(-1) == .failure(.invalidReading))
        precondition(storageCapacity(0) == .success(0))
        print("PASS: stock CPU, memory, battery, and capacity sampling")
    }
}
