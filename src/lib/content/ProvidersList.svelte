<script lang="ts">
	import type { MediaProvider } from "@/types";
	import ProviderLogo from "./ProviderLogo.svelte";

	interface Props {
		providers: MediaProvider[];
		fullListLink?: string;
		fullListLinkText?: string;
		// Used to show an "In Theaters"/"Coming Soon" message when there
		// are no providers to show yet (eg for an unreleased/theatrical-only movie).
		releaseDate?: Date;
		mediaType?: "movie" | "tv";
	}

	let {
		providers,
		fullListLink,
		fullListLinkText,
		releaseDate,
		mediaType,
	}: Props = $props();

	// How long after theatrical release a movie is still assumed to be
	// "in theaters" if we have no streaming/rent/buy providers yet.
	const IN_THEATERS_WINDOW_DAYS = 120;

	const stream = $derived(
		providers?.filter((p) => p.type === "sub" || p.type === "free") ?? [],
	);
	const rent = $derived(providers?.filter((p) => p.type === "rent") ?? []);
	const buy = $derived(providers?.filter((p) => p.type === "buy") ?? []);

	const hasProviders = $derived(providers?.length > 0);

	const releaseStatusMessage = $derived.by(() => {
		if (hasProviders || !releaseDate) return undefined;
		const now = new Date();
		if (releaseDate > now) {
			return mediaType === "tv" ? "Coming Soon" : "🎬 In Theaters Soon";
		}
		if (mediaType === "movie") {
			const daysSinceRelease =
				(now.getTime() - releaseDate.getTime()) / (1000 * 60 * 60 * 24);
			if (daysSinceRelease <= IN_THEATERS_WINDOW_DAYS) {
				return "🎬 In Theaters";
			}
		}
		return undefined;
	});
</script>

{#if hasProviders}
	<div class="providers">
		{#if stream.length > 0}
			<div class="providers-row">
				<span class="providers-row-label">Stream</span>
				<div class="providers-row-icons">
					{#each stream as provider}
						<ProviderLogo
							name={provider.name}
							logo={provider.logo}
							href={provider.link}
						/>
					{/each}
				</div>
			</div>
		{/if}
		{#if rent.length > 0}
			<div class="providers-row">
				<span class="providers-row-label">Rent</span>
				<div class="providers-row-icons">
					{#each rent as provider}
						<ProviderLogo
							name={provider.name}
							logo={provider.logo}
							href={provider.link}
						/>
					{/each}
				</div>
			</div>
		{/if}
		{#if buy.length > 0}
			<div class="providers-row">
				<span class="providers-row-label">Buy</span>
				<div class="providers-row-icons">
					{#each buy as provider}
						<ProviderLogo
							name={provider.name}
							logo={provider.logo}
							href={provider.link}
						/>
					{/each}
				</div>
			</div>
		{/if}
		{#if fullListLink}
			<!-- The fullListLink is important for TMDB data, we always show it
			 as "JustWatch" (set in component prop) because that data requires
			 attribution! but also it helps support tmdb in some way. -->
			<a class="full-list-link" href={fullListLink} target="_blank"
				>{fullListLinkText}</a
			>
		{/if}
	</div>
{:else if releaseStatusMessage}
	<div class="release-status">{releaseStatusMessage}</div>
{/if}

<style lang="scss">
	.providers {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-top: auto;
		padding-top: 10px;
	}

	.providers-row {
		display: flex;
		flex-direction: column;
		gap: 6px;

		&-label {
			font-size: 13px;
			font-weight: 600;
			color: rgba($color: #fff, $alpha: 0.6);
			text-transform: uppercase;
			letter-spacing: 0.03em;
		}

		&-icons {
			display: flex;
			align-items: center;
			flex-wrap: wrap;
			gap: 7px 15px;
		}
	}

	.full-list-link {
		align-self: flex-start;
		color: white;
		opacity: 0.7;

		&:hover {
			opacity: 1;
		}
	}

	.release-status {
		margin-top: auto;
		padding-top: 10px;
		font-size: 15px;
		font-weight: 600;
		color: rgba($color: #fff, $alpha: 0.85);
	}
</style>
