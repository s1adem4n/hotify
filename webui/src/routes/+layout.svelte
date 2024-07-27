<script lang="ts">
	import { onMount } from 'svelte';
	import '../app.css';
	import * as state from '$lib/state.svelte';

	onMount(() => {
		state.config.secret = localStorage.getItem('secret') || state.config.secret;
		state.client.secret = state.config.secret;
		state.loadServices();
	});

	$effect(() => {
		if (state.config.secret) {
			localStorage.setItem('secret', state.config.secret);
			state.client.secret = state.config.secret;
		}
	});

	let { children } = $props();
</script>

<div class="mx-auto flex w-full max-w-2xl flex-col gap-2 p-2">
	{@render children()}
</div>
