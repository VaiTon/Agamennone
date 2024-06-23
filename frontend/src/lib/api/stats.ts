export type Stats = {
    flags: number;
    queuedFlags: number;
    acceptedFlags: number;
    rejectedFlags: number;
    skippedFlags: number;
    flagsSentLastCycle?: number;
    lastCycle?: number;
};

export async function getStats(server: string) {
    const res = await fetch(server + '/api/stats');
    return (await res.json()) as Stats;
}

export type ExploitStats = Record<string, Record<string, { hour: string, count: number }[]>>


export async function getExploitStats(server: string) {
    const res = await fetch(server + '/api/stats/exploits');
    return (await res.json()) as ExploitStats;
}
