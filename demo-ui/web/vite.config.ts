import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: Number(process.env.VITE_PORT || 5177),
    strictPort: process.env.VITE_STRICT_PORT ? process.env.VITE_STRICT_PORT === 'true' : true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:3847',
        changeOrigin: true,
      },
      // Same-origin Keycloak so silent SSO iframe passes Keycloak CSP (frame-ancestors 'self').
      // Keycloak base URL in the app should be `${origin}/keycloak` (see loadAuthConfig migration).
      '/keycloak': {
        target: process.env.VITE_KEYCLOAK_PROXY_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
        configure(proxy) {
          proxy.on('proxyReq', (proxyReq, req) => {
            // Browser can send a huge Cookie header for localhost (many apps / sessions). Keycloak
            // returns 431 Request Header Fields Too Large. The 3p-cookies probe does not need those cookies.
            const path = req.url || ''
            if (path.includes('3p-cookies')) {
              proxyReq.removeHeader('cookie')
            }
          })
        },
      },
    },
  },
})
