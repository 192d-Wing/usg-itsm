import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export function Badge({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
        'bg-slate-100 text-slate-700',
        className,
      )}
    >
      {children}
    </span>
  )
}
