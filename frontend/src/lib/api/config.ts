export const getConfig = async (server: string) => {
	const res = await fetch(server + '/api/config');
	const config = await res.json();
	return config;
};
