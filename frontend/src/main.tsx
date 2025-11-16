import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { SWRConfig } from 'swr';
import './index.css';
import App from './App.tsx';
import { ThemeProvider } from '@/components/theme-provider';
import { swrConfig } from '@/lib/swr-config';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <SWRConfig value={swrConfig}>
      <ThemeProvider defaultTheme="dark" storageKey="deeb-ui-theme">
        <App />
      </ThemeProvider>
    </SWRConfig>
  </StrictMode>,
);
