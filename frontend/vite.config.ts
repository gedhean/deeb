import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import monacoEditorPlugin from 'vite-plugin-monaco-editor-esm';
import path from 'path';

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    monacoEditorPlugin({
      // Configure worker paths for Monaco Editor
      publicPath: 'monaco-editor-workers',
      customWorkers: [],
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'), // eslint-disable-line no-undef
    },
  },
  server: {
    port: 3000,
    open: true,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: true,
    exclude: ['node_modules', 'dist', 'e2e', '**/*.e2e.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/test/',
        'e2e/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData',
        '**/*.test.*',
        '**/*.spec.*',
      ],
    },
  },
});
