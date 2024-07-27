<script lang="ts">
	import ServiceCard from '$lib/components/service-card.svelte';
	import * as state from '$lib/state.svelte';
</script>

<h1 class="text-2xl font-bold">Hotify Status</h1>
<div class="flex flex-col">
	<label class="font-bold" for="secret">Secret</label>
	<input
		id="secret"
		class="rounded-xl border border-gray-100 px-4 py-2 shadow-sm focus:border-blue-500 focus:outline-none"
		type="password"
		bind:value={state.config.secret}
		placeholder="Secret"
	/>
</div>

<h2 class="mt-4 text-xl font-bold">Services</h2>
{#each state.services as service}
	<ServiceCard {service} />
{/each}
