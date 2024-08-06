<script lang="ts">
	import { type Service, ServiceStatus } from '$lib/client';
	import { client } from '$lib/state.svelte';
	import { slide } from 'svelte/transition';
	import ServiceProperty from './service-property.svelte';

	let {
		service
	}: {
		service: Service;
	} = $props();

	let logContainer: HTMLParagraphElement | null = $state(null);
	$effect(() => {
		if (logContainer) {
			logContainer.scrollTop = logContainer.scrollHeight;
		}
	});

	let open = $state(false);
</script>

<div class="flex flex-col rounded-xl border border-gray-100 px-4 py-3 shadow-sm">
	<div class="flex items-center gap-2">
		{#if service.config.proxy.match}
			<a href="https://{service.config.proxy.match}" class="font-bold hover:underline">
				{service.config.name}
			</a>
		{:else}
			<span class="font-bold">
				{service.config.name}
			</span>
		{/if}

		<button
			class="{service.status === ServiceStatus.Running
				? 'text-red-500'
				: 'text-green-500'} hover:underline"
			onclick={() => {
				if (service.status === ServiceStatus.Running) {
					client.stopService(service.config.name);
				} else {
					client.startService(service.config.name);
				}
			}}
		>
			{service.status === ServiceStatus.Running ? 'Stop' : 'Start'}
		</button>
		<button class="ml-auto" onclick={() => (open = !open)}>
			{open ? '▲' : '▼'}
		</button>
	</div>
	{#if open}
		<div
			class="mt-2 flex flex-col gap-2"
			transition:slide={{
				duration: 200
			}}
		>
			<ServiceProperty title="Repository">
				<a class="hover:underline" href={service.config.repo}>
					{service.config.repo}
				</a>
			</ServiceProperty>

			<ServiceProperty title="Run Command">
				<span class="font-mono">$ {service.config.exec}</span>
			</ServiceProperty>

			<ServiceProperty title="Build Command">
				<span class="font-mono">$ {service.config.build}</span>
			</ServiceProperty>

			<ServiceProperty title="Proxy">
				{#if service.config.proxy.match}
					<div class="flex gap-2">
						<a class="hover:underline" href="https://{service.config.proxy.match}">
							{service.config.proxy.match}
						</a>
						<span> &rarr; </span>
						<a class="hover:underline" href={service.config.proxy.upstream}>
							{service.config.proxy.upstream}
						</a>
					</div>
				{:else}
					<span>None</span>
				{/if}
			</ServiceProperty>

			<ServiceProperty title="Logs">
				<p
					class="mt-1 block max-h-48 overflow-auto whitespace-pre-line text-nowrap rounded-xl bg-gray-100 p-2 font-mono"
					bind:this={logContainer}
				>
					{service.logs.map((log) => log.trim()).join('\n')}
				</p>
			</ServiceProperty>
		</div>
	{/if}
</div>
