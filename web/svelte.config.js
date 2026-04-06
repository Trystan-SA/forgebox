import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	onwarn: (warning, handler) => {
		if (warning.code === 'css_unused_selector') return;
		handler(warning);
	},
	kit: {
		adapter: adapter({
			fallback: 'index.html'
		}),
		alias: {
			'$lib': './src/lib'
		}
	}
};

export default config;
