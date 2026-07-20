<script lang="ts">
	import Modal from "../Modal.svelte";

	interface Props {
		episodeNumber: number;
		/**
		 * If `confirmed=true` then the user wants to also mark every
		 * earlier episode in the season as finished.
		 */
		onClose: (confirmed: boolean) => void;
	}

	let { episodeNumber, onClose }: Props = $props();
</script>

<Modal
	title="Mark Previous Episodes Finished?"
	onClose={() => onClose(false)}
	maxWidth="500px"
>
	<div class="ctr">
		<p>
			Episode {episodeNumber} is finished, but some earlier episodes in this
			season aren't marked as finished yet. Do you want to also mark every
			previous episode in this season as finished?
		</p>
		<div class="btns">
			<button onclick={() => onClose(false)}>No, Just This Episode</button>
			<button class="confirm-btn" onclick={() => onClose(true)}>
				Yes, Mark Previous Episodes Finished
			</button>
		</div>
	</div>
</Modal>

<style lang="scss">
	div.ctr {
		display: flex;
		flex-flow: column;
		gap: 15px;

		.btns {
			display: flex;
			flex-flow: row wrap;
			justify-content: flex-end;
			gap: 10px;

			.confirm-btn:hover {
				color: $success;
			}
		}
	}
</style>
