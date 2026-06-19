import type { ReactNode } from 'react'
import { useAuth } from 'react-oidc-context'
import { Button } from './ui/Button'
import { Spinner } from './ui/Spinner'

// RequireAuth gates the app on a valid OIDC session, rendering a centered
// loading/login/error state until the user is authenticated.
export function RequireAuth({ children }: { children: ReactNode }) {
  const auth = useAuth()

  // Preserve the deep link the user requested across the redirect to the IdP.
  const signin = () =>
    void auth.signinRedirect({
      state: { returnTo: window.location.pathname + window.location.search },
    })

  if (auth.isLoading) {
    return <Centered><Spinner className="size-8" /></Centered>
  }

  if (auth.error) {
    return (
      <Centered>
        <div className="text-center">
          <p className="mb-3 text-sm text-rose-700">Sign-in failed: {auth.error.message}</p>
          <Button onClick={signin}>Try again</Button>
        </div>
      </Centered>
    )
  }

  if (!auth.isAuthenticated) {
    return (
      <Centered>
        <div className="text-center">
          <h1 className="mb-1 text-2xl font-semibold text-slate-800">USG-ITSM</h1>
          <p className="mb-4 text-sm text-slate-500">Sign in to continue.</p>
          <Button onClick={signin}>Sign in</Button>
        </div>
      </Centered>
    )
  }

  return <>{children}</>
}

function Centered({ children }: { children: ReactNode }) {
  return <div className="flex h-full items-center justify-center">{children}</div>
}
