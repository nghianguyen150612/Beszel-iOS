export interface ContainerRowSnapshot {
	id: string
	system: string
	updated: number
}

export const containerRefreshWindowMs = 70_000

export function selectRecentContainers<T extends ContainerRowSnapshot>(items: T[]): T[] {
	if (items.length === 0) return []
	const lastUpdated = Math.max(items[0].updated, items.at(-1)?.updated ?? 0)
	return items.filter((item) => Math.abs(lastUpdated - item.updated) < containerRefreshWindowMs)
}

/** Replace only one system's refreshed snapshot, retaining the existing short stale-row grace period. */
export function mergeSystemContainers<T extends ContainerRowSnapshot>(
	current: T[],
	systemId: string,
	items: T[]
): T[] {
	if (items.length === 0) return current.filter((item) => item.system !== systemId)

	const lastUpdated = Math.max(items[0].updated, items.at(-1)?.updated ?? 0)
	const containerIds = new Set<string>()
	const next: T[] = []
	for (const item of items) {
		if (item.system === systemId && Math.abs(lastUpdated - item.updated) < containerRefreshWindowMs) {
			containerIds.add(item.id)
			next.push(item)
		}
	}
	for (const item of current) {
		if (item.system !== systemId || (!containerIds.has(item.id) && lastUpdated - item.updated < containerRefreshWindowMs)) {
			next.push(item)
		}
	}
	return next
}

type ContainerRefreshOptions<T extends ContainerRowSnapshot> = {
	fetch: (systemId: string | undefined, signal: AbortSignal) => Promise<T[]>
	onInitial: (items: T[]) => void
	onSystem: (systemId: string, items: T[]) => void
	onSystemRemoved: (systemId: string) => void
	onError: (error: unknown) => void
}

/** Owns request cancellation and response ordering for one mounted containers table. */
export class ContainerRefreshCoordinator<T extends ContainerRowSnapshot> {
	private active = true
	private initialComplete = false
	private initialStarted = false
	private queuedSystems = new Set<string>()
	private removedSystems = new Set<string>()
	private generations = new Map<string | undefined, number>()
	private controllers = new Map<string | undefined, AbortController>()

	constructor(private readonly options: ContainerRefreshOptions<T>) {}

	startAll() {
		if (!this.active || this.initialStarted) return
		this.initialStarted = true
		this.initialComplete = false
		void this.request(undefined).then(() => {
			if (!this.active) return
			this.initialComplete = true
			const queued = [...this.queuedSystems]
			this.queuedSystems.clear()
			for (const systemId of queued) this.refreshSystem(systemId)
		})
	}

	startSystem(systemId: string) {
		if (!this.active) return
		this.initialStarted = true
		this.initialComplete = true
		this.removedSystems.delete(systemId)
		void this.request(systemId)
	}

	refreshSystem(systemId: string) {
		if (!this.active) return
		this.removedSystems.delete(systemId)
		if (!this.initialComplete) {
			this.queuedSystems.add(systemId)
			return
		}
		void this.request(systemId)
	}

	removeSystem(systemId: string) {
		this.queuedSystems.delete(systemId)
		this.removedSystems.add(systemId)
		this.cancelRequest(systemId)
		this.options.onSystemRemoved(systemId)
	}

	dispose() {
		if (!this.active) return
		this.active = false
		this.queuedSystems.clear()
		this.removedSystems.clear()
		for (const controller of this.controllers.values()) controller.abort()
		this.controllers.clear()
		this.generations.clear()
	}

	private cancelRequest(systemId: string | undefined) {
		this.generations.set(systemId, (this.generations.get(systemId) ?? 0) + 1)
		this.controllers.get(systemId)?.abort()
		this.controllers.delete(systemId)
	}

	private async request(systemId: string | undefined) {
		if (!this.active) return
		const generation = (this.generations.get(systemId) ?? 0) + 1
		this.generations.set(systemId, generation)
		this.controllers.get(systemId)?.abort()
		const controller = new AbortController()
		this.controllers.set(systemId, controller)

		try {
			const items = await this.options.fetch(systemId, controller.signal)
			if (!this.active || controller.signal.aborted || this.generations.get(systemId) !== generation) return
			if (systemId === undefined) {
				this.options.onInitial(items.filter((item) => !this.removedSystems.has(item.system)))
			}
			else this.options.onSystem(systemId, items)
		} catch (error) {
			if (this.active && !controller.signal.aborted && !isAbortError(error) && this.generations.get(systemId) === generation) {
				this.options.onError(error)
			}
		} finally {
			if (this.controllers.get(systemId) === controller) this.controllers.delete(systemId)
		}
	}
}

function isAbortError(error: unknown) {
	return Boolean(
		error &&
		typeof error === "object" &&
		(("isAbort" in error && error.isAbort) || ("name" in error && error.name === "AbortError"))
	)
}
