export type Flag = {
    flag: string;
    sploit: string;
    team: string;
    receivedTime: string;
    sentTime: string | null;
    status: string;
    checkSystemResponse: string | null;
};

export async function getFlags(server: string) {
    const limit = 50;
    const res = await fetch(`${server}/api/flags?limit=${limit}`);
    const flags: Flag[] = await res.json();

    return flags;
}
