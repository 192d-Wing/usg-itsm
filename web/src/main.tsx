import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { AuthProvider } from 'react-oidc-context'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { makeOidcConfig } from './auth/oidc'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
})

// Runtime config served by the gateway (/config.json) keeps the built bundle
// environment-agnostic. In dev (no gateway) fall back to build-time env.
async function loadConfig(): Promise<{ authority: string; clientId: string }> {
  try {
    const res = await fetch('/config.json', { cache: 'no-store' })
    if (res.ok) {
      const c = (await res.json()) as { oidcAuthority?: string; oidcClientId?: string }
      if (c.oidcAuthority && c.oidcClientId) {
        return { authority: c.oidcAuthority, clientId: c.oidcClientId }
      }
    }
  } catch {
    // fall through to build-time env
  }
  return {
    authority: import.meta.env.VITE_OIDC_AUTHORITY,
    clientId: import.meta.env.VITE_OIDC_CLIENT_ID,
  }
}

const rootEl = document.getElementById('root')
if (!rootEl) throw new Error('root element not found')

void loadConfig().then(({ authority, clientId }) => {
  createRoot(rootEl).render(
    <StrictMode>
      <AuthProvider {...makeOidcConfig(authority, clientId)}>
        <QueryClientProvider client={queryClient}>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </QueryClientProvider>
      </AuthProvider>
    </StrictMode>,
  )
})
