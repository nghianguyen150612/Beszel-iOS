import { expect, test } from "bun:test"

test("stores load and toggle log timestamps when browser storage is unavailable", async () => {
	const originalStorage = globalThis.localStorage
	Object.defineProperty(globalThis, "localStorage", { configurable: true, value: undefined })
	try {
		const { $showLogTimestamps, toggleLogTimestamps } = await import("./stores")
		expect($showLogTimestamps.get()).toBe(true)
		expect(() => toggleLogTimestamps()).not.toThrow()
		expect($showLogTimestamps.get()).toBe(false)
	} finally {
		Object.defineProperty(globalThis, "localStorage", { configurable: true, value: originalStorage })
	}
})
