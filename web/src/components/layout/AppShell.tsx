import type { ReactNode } from 'react'
import { useAuth } from 'react-oidc-context'
import { NavLink } from 'react-router-dom'
import { cn } from '../../lib/cn'
import { isAgent, rolesFromToken } from '../../lib/jwt'
import { Button } from '../ui/Button'

export function AppShell({ children }: { children: ReactNode }) {
  const auth = useAuth()
  const roles = rolesFromToken(auth.user?.access_token)
  const profile = auth.user?.profile
  const name = profile?.name ?? profile?.preferred_username ?? profile?.email ?? 'User'

  return (
    <div className="flex h-full">
      <aside className="flex w-60 flex-col border-r border-slate-200 bg-white">
        <div className="px-5 py-4 text-lg font-semibold text-slate-800">
          USG<span className="text-brand-600">-ITSM</span>
        </div>
        <nav className="flex-1 space-y-1 px-3">
          <NavItem to="/" end>Queue</NavItem>
          <NavItem to="/new">New Ticket</NavItem>
        </nav>
        <div className="border-t border-slate-100 p-3 text-xs text-slate-500">
          <div className="truncate font-medium text-slate-700">{name}</div>
          <div className="mb-2">{isAgent(roles) ? 'Agent' : 'Requester'}</div>
          <Button variant="secondary" size="sm" className="w-full" onClick={() => void auth.signoutRedirect()}>
            Sign out
          </Button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto">
        <div className="mx-auto max-w-5xl p-6">{children}</div>
      </main>
    </div>
  )
}

function NavItem({ to, end, children }: { to: string; end?: boolean; children: ReactNode }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          'block rounded-md px-3 py-2 text-sm font-medium',
          isActive ? 'bg-brand-50 text-brand-700' : 'text-slate-600 hover:bg-slate-100',
        )
      }
    >
      {children}
    </NavLink>
  )
}
