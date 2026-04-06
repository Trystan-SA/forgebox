import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	onwarn: (warning, handler) => {
		// Suppress false positives from SCSS nesting with &__
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
