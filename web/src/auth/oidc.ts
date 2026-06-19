import { WebStorageStateStore, type User } from 'oidc-client-ts'
import type { AuthProviderProps } from 'react-oidc-context'

// makeOidcConfig builds the OIDC (Authorization Code + PKCE) settings for the
// public SPA client. IdP-agnostic: authority/clientId come from runtime config
// so one built bundle works across deployments.
export function makeOidcConfig(authority: string, clientId: string): AuthProviderProps {
  return {
    authority,
    client_id: clientId,
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
}

