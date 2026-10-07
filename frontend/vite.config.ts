import { defineConfig, loadEnv, searchForWorkspaceRoot } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig(({ mode }) => {
  // Load env file from project root (one level up from frontend)
  const environment = loadEnv(mode, path.resolve(__dirname, '..'), '')
  const backendPort = (environment.APP_ENV === 'dev' && environment.APP_PORT_DEV) || environment.APP_PORT || '8080'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
        '@sepay': path.resolve(__dirname, '../pkg/sepay/web'),
      },
    },
    server: {
      fs: {
        allow: [searchForWorkspaceRoot(process.cwd()), path.resolve(__dirname, '../pkg/sepay/web')],
      },
      allowedHosts: [
        environment.APP_URL,
        environment.APP_URL_DEV
      ].filter(Boolean) as string[],
      proxy: {
        '/api': {
          target: `http://localhost:${backendPort}`,
          changeOrigin: true,
          xfwd: true,
        }
      }
    }
  }
})
