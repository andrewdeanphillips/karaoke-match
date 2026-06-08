import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // Bind to the IPv4 loopback specifically — Spotify's OAuth redirect_uri
    // is registered against 127.0.0.1, and browsers treat that as a
    // different site than localhost (which otherwise resolves to ::1) for
    // cookie purposes. The dev server and the API need to agree on one.
    host: '127.0.0.1',
  },
})
