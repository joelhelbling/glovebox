import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';

// https://astro.build/config
export default defineConfig({
  site: 'https://joelhelbling.github.io',
  base: '/glovebox',
  integrations: [tailwind()],
  output: 'static',
});
