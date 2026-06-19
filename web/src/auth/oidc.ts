import { WebStorageStateStore, type User } from 'oidc-client-ts'
import type { AuthProviderProps } from 'react-oidc-context'

// OIDC (Authorization Code + PKCE) for the public SPA client. IdP-agnostic:
// any OIDC provider works via VITE_OIDC_AUTHORITY.
export const oidcConfig: AuthProviderProps = {
  authority: import.meta.env.VITE_OIDC_AUTHORITY,
  client_id: import.meta.env.VITE_OIDC_CLIENT_ID,
  redirect_uri: window.location.origin + '/',
  post_logout_redirect_uri: window.location.origin + '/',
  scope: 'openid profile email',
  userStore: new WebStorageStateStore({ store: window.localStorage }),
  // After sign-in, strip the auth code/state from the URL and restore the
  // deep link the user originally requested (carried in the auth state).
  onSigninCallback: (user?: User) => {
    const state = user?.state as { returnTo?: string } | undefined
    window.history.replaceState({}, document.title, state?.returnTo ?? window.location.pathname)
  },
}

