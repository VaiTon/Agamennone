export type Flag = {
	flag: string;
	sploit: string;
	team: string;
	receivedTime: string;
	status: string;
	checkSystemResponse: string | null;
	sentCycle: number | null;
};

export async function getFlags(server: string) {
	const flags = (await (await fetch(server + '/api/flags')).json()) as Flag[];
	flags.reverse();
	return flags;
}
