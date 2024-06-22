export async function getStats(server: string) {
	const res = await fetch(server + '/api/stats');
	const stats = (await res.json()) as Stats;
	return stats;
}

export type Stats = {
	flags: number;
	queuedFlags: number;
	acceptedFlags: number;
	rejectedFlags: number;
	skippedFlags: number;
	flagsSentLastCycle?: number;
	lastCycle?: number;
};
