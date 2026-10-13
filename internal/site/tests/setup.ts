const values = new Map<string, string>()

if (typeof globalThis.localStorage === "undefined") {
	Object.defineProperty(globalThis, "localStorage", {
		configurable: true,
		value: {
			get length() {
				return values.size
			},
			clear() {
				values.clear()
			},
			getItem(key: string) {
				return values.get(key) ?? null
			},
			key(index: number) {
				return [...values.keys()][index] ?? null
			},
			removeItem(key: string) {
				values.delete(key)
			},
			setItem(key: string, value: string) {
				values.set(key, String(value))
			},
		},
	})
}
