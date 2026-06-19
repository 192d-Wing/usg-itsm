import { useState } from 'react'
import { useAuth } from 'react-oidc-context'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import type { Status } from '../../api/types'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Input, Label, Select, Textarea } from '../../components/ui/Field'
import { Spinner } from '../../components/ui/Spinner'
import { formatDateTime } from '../../lib/format'
import { isAgent, rolesFromToken } from '../../lib/jwt'
import {
  useAddComment,
  useComments,
  useTicket,
  useTransition,
  useUpdateTicket,
} from './hooks'
import {
  PRIORITY_CLASS,
  PRIORITY_LABEL,
  STATUS_CLASS,
  STATUS_LABEL,
  TRANSITIONS,
  TYPE_LABEL,
} from './meta'

export function TicketDetailPage() {
  const { id = '' } = useParams()
  const auth = useAuth()
  const agent = isAgent(rolesFromToken(auth.user?.access_token))

  const ticket = useTicket(id)
  const comments = useComments(id)

  if (ticket.isLoading) {
    return (
      <div className="flex justify-center p-10">
        <Spinner className="size-8" />
      </div>
    )
  }
  if (ticket.isError || !ticket.data) {
    const msg = ticket.error instanceof ApiError ? ticket.error.message : 'Ticket not found.'
    return (
      <div>
        <Link to="/" className="text-sm text-brand-700 hover:underline">
          ← Back to queue
        </Link>
        <p className="mt-4 text-sm text-rose-700">{msg}</p>
      </div>
    )
  }

  const t = ticket.data

  return (
    <div>
      <Link to="/" className="text-sm text-brand-700 hover:underline">
        ← Back to queue
      </Link>

      <div className="mb-4 mt-2 flex items-start justify-between gap-4">
        <div>
          <div className="font-mono text-xs text-slate-500">{t.number}</div>
          <h1 className="text-xl font-semibold text-slate-800">{t.title}</h1>
        </div>
        <div className="flex gap-2">
          <Badge className={PRIORITY_CLASS[t.priority]}>{PRIORITY_LABEL[t.priority]}</Badge>
          <Badge className={STATUS_CLASS[t.status]}>{STATUS_LABEL[t.status]}</Badge>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <Card>
            <CardHeader title="Description" />
            <CardBody>
              <p className="whitespace-pre-wrap text-sm text-slate-700">
                {t.description || <span className="text-slate-400">No description.</span>}
              </p>
            </CardBody>
          </Card>

          <CommentsCard
            id={id}
            agent={agent}
            loading={comments.isLoading}
            isError={comments.isError}
            items={comments.data?.items ?? []}
          />
        </div>

        <div className="space-y-4">
          <Card>
            <CardHeader title="Details" />
            <CardBody className="space-y-2 text-sm">
              <Detail label="Type" value={TYPE_LABEL[t.type]} />
              <Detail label="Requester" value={t.requester_id} />
              <Detail label="Assignee" value={t.assignee_id || '—'} />
              <Detail label="Group" value={t.assignment_group || '—'} />
              <Detail label="Created" value={formatDateTime(t.created_at)} />
              <Detail label="Updated" value={formatDateTime(t.updated_at)} />
            </CardBody>
          </Card>

          {/* key resets local edit state when the server-side assignee changes */}
          {agent && (
            <AgentActions
              key={t.assignee_id ?? 'unassigned'}
              id={id}
              status={t.status}
              assignee={t.assignee_id ?? ''}
            />
          )}
        </div>
      </div>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-slate-500">{label}</span>
      <span className="truncate font-medium text-slate-700">{value}</span>
    </div>
  )
}

function AgentActions({ id, status, assignee }: { id: string; status: Status; assignee: string }) {
  const transition = useTransition(id)
  const update = useUpdateTicket(id)
  const [next, setNext] = useState<Status | ''>('')
  const [assigneeId, setAssigneeId] = useState(assignee)
  const options = TRANSITIONS[status]

  return (
    <Card>
      <CardHeader title="Actions" />
      <CardBody className="space-y-4">
        <div>
          <Label htmlFor="transition">Change status</Label>
          <div className="flex gap-2">
            <Select
              id="transition"
              value={next}
              disabled={options.length === 0}
              onChange={(e) => setNext(e.target.value as Status | '')}
            >
              <option value="">{options.length ? 'Select…' : 'No transitions'}</option>
              {options.map((s) => (
                <option key={s} value={s}>
                  {STATUS_LABEL[s]}
                </option>
              ))}
            </Select>
            <Button
              disabled={!next || transition.isPending}
              onClick={() => next && transition.mutate({ status: next }, { onSuccess: () => setNext('') })}
            >
              Apply
            </Button>
          </div>
          {transition.isError && (
            <p className="mt-1 text-xs text-rose-700">
              {transition.error instanceof ApiError ? transition.error.message : 'Transition failed.'}
            </p>
          )}
        </div>

        <div>
          <Label htmlFor="assignee">Assignee</Label>
          <div className="flex gap-2">
            <Input
              id="assignee"
              value={assigneeId}
              onChange={(e) => setAssigneeId(e.target.value)}
              placeholder="user id"
            />
            <Button
              variant="secondary"
              disabled={update.isPending}
              onClick={() => update.mutate({ assignee_id: assigneeId })}
            >
              Save
            </Button>
          </div>
          {update.isError && (
            <p className="mt-1 text-xs text-rose-700">
              {update.error instanceof ApiError ? update.error.message : 'Assignment failed.'}
            </p>
          )}
        </div>
      </CardBody>
    </Card>
  )
}

function CommentsCard({
  id,
  agent,
  loading,
  isError,
  items,
}: {
  id: string
  agent: boolean
  loading: boolean
  isError: boolean
  items: { id: string; author_id: string; body: string; internal: boolean; created_at: string }[]
}) {
  const add = useAddComment(id)
  const [body, setBody] = useState('')
  const [internal, setInternal] = useState(false)

  return (
    <Card>
      <CardHeader title="Activity" />
      <CardBody className="space-y-4">
        <CommentList loading={loading} isError={isError} items={items} />

        <form
          className="space-y-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (!body.trim()) return
            add.mutate({ body, internal }, { onSuccess: () => setBody('') })
          }}
        >
          <Textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Add a comment…"
          />
          <div className="flex items-center justify-between">
            {agent ? (
              <label className="flex items-center gap-2 text-sm text-slate-600">
                <input
                  type="checkbox"
                  checked={internal}
                  onChange={(e) => setInternal(e.target.checked)}
                />
                Internal note
              </label>
            ) : (
              <span />
            )}
            <Button type="submit" size="sm" disabled={add.isPending || !body.trim()}>
              Comment
            </Button>
          </div>
          {add.isError && (
            <p className="text-xs text-rose-700">
              {add.error instanceof ApiError ? add.error.message : 'Failed to add comment.'}
            </p>
          )}
        </form>
      </CardBody>
    </Card>
  )
}

function CommentList({
  loading,
  isError,
  items,
}: {
  loading: boolean
  isError: boolean
  items: { id: string; author_id: string; body: string; internal: boolean; created_at: string }[]
}) {
  if (loading) return <Spinner />
  if (isError) return <p className="text-sm text-rose-700">Failed to load activity.</p>
  if (items.length === 0) return <p className="text-sm text-slate-500">No comments yet.</p>
  return (
    <ul className="space-y-3">
      {items.map((c) => (
        <li key={c.id} className="rounded-md border border-slate-100 p-3">
          <div className="mb-1 flex items-center gap-2 text-xs text-slate-500">
            <span className="font-medium text-slate-700">{c.author_id}</span>
            <span>{formatDateTime(c.created_at)}</span>
            {c.internal && <Badge className="bg-amber-100 text-amber-800">Internal</Badge>}
          </div>
          <p className="whitespace-pre-wrap text-sm text-slate-700">{c.body}</p>
        </li>
      ))}
    </ul>
  )
}
