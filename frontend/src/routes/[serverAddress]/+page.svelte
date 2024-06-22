<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import type { PageData } from './$types';
	import { getFlags, type Flag } from '$lib/api/flags';
	import { getConfig } from '$lib/api/config';
	import dayjs from 'dayjs';
	import relativeTime from 'dayjs/plugin/relativeTime';
	import { getStats, type Stats } from '$lib/api/stats';
	dayjs.extend(relativeTime);

	export let data: PageData;

	let flags: Flag[] | undefined = undefined;
	let config: object | undefined = undefined;
	let stats: Stats | undefined = undefined;

	let errored = false;

	const refresh = async () => {
		try {
			await fetch(data.serverAddress + '/');
			errored = false;
		} catch (e) {
			errored = true;
			return;
		}

		flags = await getFlags(data.serverAddress);
		config = await getConfig(data.serverAddress);
		stats = await getStats(data.serverAddress);
	};

	const DEFAULT_INTERVAL = 10;

	let refreshInterval = DEFAULT_INTERVAL;
	let timerId: number;
	onMount(async () => {
		await refresh();
		timerId = setInterval(() => {
			refreshInterval--;
			if (refreshInterval <= 0) {
				refreshInterval = DEFAULT_INTERVAL;
				refresh();
			}
		}, 1000);
	});

	onDestroy(() => {
		clearInterval(timerId);
	});

	const BADGE_STATUS_MAP = new Map<string, string>([
		['ACCEPTED', 'success'],
		['REJECTED', 'error'],
		['SKIPPED', 'warning'],
		['QUEUED', '']
	]);
</script>

<nav
	class="navbar bg-secondary text-primary-content rounded-xl shadow-md mb-8 mt-8 max-w-5xl mx-auto sticky top-2 z-10"
>
	<div class="navbar-start">
		<a class="btn btn-ghost" href="/">Change server</a>
	</div>
	<div class="navbar-center">
		<span class="font-mono">
			{#if refreshInterval == DEFAULT_INTERVAL}
				Refreshing...
			{:else}
				Connected to {data.serverAddress}
			{/if}
		</span>
	</div>
	<div class="navbar-end">
		<button
			class="countdown me-4 text-xl font-mono btn btn-ghost"
			title="Refresh now"
			on:click={() => {
				refreshInterval = DEFAULT_INTERVAL;
				refresh();
			}}
		>
			<span style="--value:{refreshInterval};" />
		</button>
	</div>
</nav>

{#if errored}
	<div class="alert alert-error shadow-lg max-w-3xl mx-auto">
		<div>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="stroke-current flex-shrink-0 h-6 w-6"
				fill="none"
				viewBox="0 0 24 24"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
				/></svg
			>
			<span>Error while fetching data</span>
		</div>
	</div>
{:else}
	{#if stats != null}
		<div class="flex gap-10 mx-10">
			<div class="card bg-base-200 w-96 grow">
				<p class="card-body text-xl font-mono text-center">
					Flags sent last cycle: {stats.flagsSentLastCycle}
				</p>
			</div>
			<div class="card bg-base-200 w-96 grow">
				<div class="card-body text-xl font-mono text-center">
					<p>Queued flags: {stats.queuedFlags}</p>
					<p>Accepted flags: {stats.acceptedFlags}</p>
					<p>Rejected flags: {stats.rejectedFlags}</p>
					<p>Skipped flags: {stats.skippedFlags}</p>
				</div>
			</div>
			<div class="card bg-base-200 w-96 grow">
				<p class="card-body text-xl font-mono text-center">Cycle: {stats.lastCycle}</p>
			</div>
			<div class="card bg-base-200 w-96 grow">
				<p class="card-body text-xl font-mono text-center">A-Index: 0.00</p>
			</div>
		</div>
	{/if}

	<div class="max-w-6xl mx-auto my-10">
		{#if flags != null}
			<h2 class="font-bold text-xl text-center mb-2">Flags</h2>
			<div class="overflow-auto max-h-[400px] bg-base-300 rounded">
				<table class="table table-compact w-full">
					<thead class="sticky top-0">
						<td>Team</td>
						<td>Flag</td>
						<td>Status</td>
						<td>Resp</td>
						<td>Exploit</td>
						<td>Received</td>
						<td>Sent</td>
					</thead>

					{#each flags as flag}
						{@const badgeClass = BADGE_STATUS_MAP.get(flag.status)}

						<tr class="hover" transition:fade>
							<td>{flag.team}</td>
							<td class="font-mono">{flag.flag}</td>
							<td>
								<div
									class="badge"
									class:badge-success={badgeClass == 'success'}
									class:badge-error={badgeClass == 'error'}
									class:badge-warning={badgeClass == 'warning'}
								>
									{flag.status}
								</div>
							</td>
							<td>{flag.checkSystemResponse}</td>
							<td class="font-mono">{flag.sploit}</td>
							<td title={flag.receivedTime}>
								{dayjs(flag.receivedTime).fromNow()}
							</td>
							<td>{flag.sentCycle}</td>
						</tr>
					{/each}
				</table>
			</div>
		{/if}

		{#if config != null}
			<h2 class="font-bold text-xl text-center mb-2 mt-10">Config</h2>
			<div>
				<pre class="bg-base-300 rounded max-h-[400px] overflow-y-auto p-2">
{JSON.stringify(config, null, 2)}
				</pre>
			</div>
		{/if}
	</div>
{/if}
