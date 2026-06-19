import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError } from '../../api/client'
import type { Status, WorkItemType } from '../../api/types'
import { Badge } from '../../components/ui/Badge'
import { Card } from '../../components/ui/Card'
import { Select } from '../../components/ui/Field'
import { Spinner } from '../../components/ui/Spinner'
import { formatDateTime } from '../../lib/format'
import { useTickets } from './hooks'
import {
  PRIORITY_CLASS,
  PRIORITY_LABEL,
  STATUS_CLASS,
  STATUS_LABEL,
  TYPE_LABEL,
} from './meta'

export function TicketsPage() {
  const [type, setType] = useState<WorkItemType | ''>('')
  const [status, setStatus] = useState<Status | ''>('')
  const query = useTickets({ type: type || undefined, status: status || undefined })

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-xl font-semibold text-slate-800">Queue</h1>
        <div className="flex gap-2">
          <Select
            aria-label="Filter by type"
            className="w-44"
            value={type}
            onChange={(e) => setType(e.target.value as WorkItemType | '')}
          >
            <option value="">All types</option>
            <option value="incident">Incidents</option>
            <option value="service_request">Service Requests</option>
          </Select>
          <Select
            aria-label="Filter by status"
            className="w-40"
            value={status}
            onChange={(e) => setStatus(e.target.value as Status | '')}
          >
            <option value="">All statuses</option>
            {(Object.keys(STATUS_LABEL) as Status[]).map((s) => (
              <option key={s} value={s}>
                {STATUS_LABEL[s]}
              </option>
            ))}
          </Select>
        </div>
      </div>

      <Card>
        {query.isLoading ? (
          <div className="flex justify-center p-10">
            <Spinner className="size-8" />
          </div>
        ) : query.isError ? (
          <p className="p-6 text-sm text-rose-700">
            {query.error instanceof ApiError ? query.error.message : 'Failed to load tickets.'}
          </p>
        ) : query.data && query.data.items.length > 0 ? (
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-2 font-medium">Number</th>
                <th className="px-4 py-2 font-medium">Title</th>
                <th className="px-4 py-2 font-medium">Type</th>
                <th className="px-4 py-2 font-medium">Priority</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Updated</th>
              </tr>
            </thead>
            <tbody>
              {query.data.items.map((t) => (
                <tr key={t.id} className="border-b border-slate-50 last:border-0 hover:bg-slate-50">
                  <td className="px-4 py-2 font-mono text-xs text-slate-500">
                    <Link to={`/tickets/${t.id}`} className="text-brand-700 hover:underline">
                      {t.number}
                    </Link>
                  </td>
                  <td className="px-4 py-2">
                    <Link to={`/tickets/${t.id}`} className="font-medium text-slate-800 hover:underline">
                      {t.title}
                    </Link>
                  </td>
                  <td className="px-4 py-2 text-slate-600">{TYPE_LABEL[t.type]}</td>
                  <td className="px-4 py-2">
                    <Badge className={PRIORITY_CLASS[t.priority]}>{PRIORITY_LABEL[t.priority]}</Badge>
                  </td>
                  <td className="px-4 py-2">
                    <Badge className={STATUS_CLASS[t.status]}>{STATUS_LABEL[t.status]}</Badge>
                  </td>
                  <td className="px-4 py-2 text-slate-500">{formatDateTime(t.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="p-6 text-sm text-slate-500">No tickets match your filters.</p>
        )}
      </Card>
    </div>
  )
}
