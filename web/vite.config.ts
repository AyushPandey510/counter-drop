import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'
import { fileURLToPath, URL } from 'node:url'

// In dev, the API runs on :8080 and Vite proxies /api to it, so phones on the LAN
// only need to reach the Vite server (npm run dev -- --host).
export default defineConfig({
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    port: 5173,
    proxy: { '/api': { target: process.env.VITE_API_PROXY ?? 'http://localhost:8080', changeOrigin: true } },
  },
  build: { target: 'es2020', chunkSizeWarningLimit: 1500 },
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['icon.svg'],
      manifest: {
        name: 'Counter Drop',
        short_name: 'Counter Drop',
        description: 'Send files to the print counter. No WhatsApp, deleted after pickup.',
        start_url: '/?src=pwa',
        scope: '/',
        display: 'standalone',
        background_color: '#F8FAFC',
        theme_color: '#0F172A',
        icons: [
          { src: 'icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
        maximumFileSizeToCacheInBytes: 3 * 1024 * 1024,
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//],
        runtimeCaching: [],
      },
    }),
  ],
})
