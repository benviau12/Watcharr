<!-- Renders a watch provider's official logo, as supplied by tmdb. -->
<script lang="ts">
	interface Props {
		name: string;
		logo?: string;
		href?: string;
		wh?: number;
	}

	let { name, logo, href, wh = 40 }: Props = $props();

	// tmdb rarely gives us a direct link for movie/tv providers, so only
	// render an anchor when we actually have somewhere to send the user.
	const tag = $derived(href ? "a" : "span");
</script>

<svelte:element
	this={tag}
	class="provider-logo"
	class:provider-logo-text={!logo}
	aria-label={name}
	title={name}
	{href}
	target={href ? "_blank" : undefined}
>
	{#if logo}
		<img
			src={`https://image.tmdb.org/t/p/w92${logo}`}
			alt={name}
			width={wh}
			height={wh}
			loading="lazy"
		/>
	{:else}
		{name}
	{/if}
</svelte:element>

<style lang="scss">
	.provider-logo {
		display: flex;
		opacity: 0.85;
		transition: opacity 0.15s ease;

		&:hover {
			opacity: 1;
		}

		img {
			border-radius: 6px;
			object-fit: cover;
		}
	}

	.provider-logo-text {
		align-items: center;
		color: white;
		font-size: 13px;
		font-weight: 600;
		text-decoration: none;
		background: rgba(255, 255, 255, 0.1);
		border-radius: 6px;
		padding: 6px 10px;
	}
</style>
