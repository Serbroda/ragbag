/** Size of the uploaded PNG; the server scales space icons to 256 px. */
const SIZE = 512;

export function randomPlanetSeed(): string {
	// Not crypto.randomUUID(): that needs HTTPS, and plain http in the LAN is fine for ragbag
	return Math.random().toString(36).slice(2);
}

/** The planet for a seed as a data URI, for previews. Loads the generator on first use (it is not small). */
export async function planetDataUri(seed: string): Promise<string> {
	const [{ Avatar }, planets] = await Promise.all([
		import('@dicebear/core'),
		// "Planets" by DiceBear (https://www.dicebear.com), CC0 1.0: generated in the browser, nothing is sent anywhere
		import('@dicebear/styles/planets.json'),
	]);
	return new Avatar(planets.default, { seed, size: SIZE }).toDataUri();
}

/** The planet as a PNG file to upload like any other space icon (the server takes no SVG). */
export async function planetFile(seed: string): Promise<File> {
	const image = new Image(SIZE, SIZE);
	image.src = await planetDataUri(seed);
	await image.decode();
	const canvas = Object.assign(document.createElement('canvas'), { width: SIZE, height: SIZE });
	canvas.getContext('2d')!.drawImage(image, 0, 0, SIZE, SIZE);
	const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'));
	if (!blob) throw new Error('The planet could not be drawn.');
	return new File([blob], 'planet.png', { type: 'image/png' });
}
