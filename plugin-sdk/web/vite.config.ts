import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

const pluginDir = process.env.PLUGIN_DIR
if (!pluginDir) throw new Error('Thiếu PLUGIN_DIR, ví dụ: PLUGIN_DIR=examples/debt-reminder')
const root = path.resolve(pluginDir)

const shared = {
  react: '__qlpt_react__',
  'react/jsx-runtime': '__qlpt_jsx__',
  'react-dom': '__qlpt_react_dom__',
  '@qlpt/plugin': '__qlpt_sdk__',
  '@qlpt/plugin/ui': '__qlpt_ui__',
  '@qlpt/plugin/icons': '__qlpt_icons__',
}

export default defineConfig({
  root,
  publicDir: false,
  plugins: [tailwindcss()],
  build: {
    outDir: path.join(root, 'build/web'),
    emptyOutDir: true,
    target: 'es2022',
    cssCodeSplit: false,
    lib: {
      entry: path.join(root, 'web/index.tsx'),
      formats: ['iife'],
      name: '__qlpt_plugin__',
      fileName: () => 'index.js',
      cssFileName: 'index',
    },
    rollupOptions: {
      external: Object.keys(shared),
      output: { globals: shared, exports: 'named' },
    },
  },
})
