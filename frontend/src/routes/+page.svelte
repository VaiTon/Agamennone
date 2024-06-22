<script lang="ts">
	let address: string;
</script>

<div class="mx-auto w-max my-10">
	<div class="form-control">
		<label class="input-group">
			<span class="label-text"> Server address: </span>
			<input class="input input-bordered" bind:value={address} />
			<a href={btoa(address)} class="btn btn-primary"> Submit </a>
		</label>
	</div>
</div>
