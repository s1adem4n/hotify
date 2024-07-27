<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		children
	}: {
		title: string;
		children: Snippet;
	} = $props();
</script>

<div class="flex flex-col">
	<span class="font-bold">
		{title}
	</span>
	{@render children()}
</div>
