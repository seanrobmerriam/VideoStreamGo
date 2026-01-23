import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import tailwind from '@astrojs/tailwind';
import path from 'path';

export default defineConfig({
  integrations: [react(), tailwind()],
  server: {
    port: 3000,
    host: true,
  },
  vite: {
    ssr: {
      noExternal: ['react', 'react-dom'],
    },
    resolve: {
      alias: {
        '@': path.resolve('./src'),
        '@components': path.resolve('./src/components'),
        '@layouts': path.resolve('./src/layouts'),
        '@pages': path.resolve('./src/pages'),
        '@lib': path.resolve('./src/lib'),
        '@stores': path.resolve('./src/stores'),
      },
    },
  },
  astroTypescript: {
    strict: false,
  },
});
