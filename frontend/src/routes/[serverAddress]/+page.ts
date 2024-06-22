import type { PageLoad } from './$types';

export const ssr = false;

export const load = (({ params }) => {
	const serverAddress = atob(params.serverAddress);
	return { serverAddress };
}) satisfies PageLoad;
