import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.ico', 'robots.txt'],
      manifest: {
        name: 'Public Accountability Platform',
        short_name: 'Accountability',
        description: 'Voice-first public accountability platform',
        theme_color: '#1976d2',
        icons: [
          {
            src: 'icon-192.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'any'
          },
          {
            src: 'icon-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any'
          }
        ]
      },
      workboxOptions: {
        skipWaiting: true,
        clientsClaim: true
      },
      // Suppress errors for missing icons during development
      devOptions: {
        enabled: false, // Disable PWA in dev to avoid icon errors
        type: 'module'
      }
    })
  ],
  server: {
    port: 3000
  }
});
