import { api } from './client'
import type {
  Comment,
  CreateWorkItem,
  ListParams,
  Priority,
  Status,
  WorkItem,
} from './types'

function query(params: ListParams): string {
  const sp = new URLSearchParams()
  if (params.type) sp.set('type', params.type)
  if (params.status) sp.set('status', params.status)
  if (params.assignee) sp.set('assignee', params.assignee)
  if (params.limit != null) sp.set('limit', String(params.limit))
  if (params.offset != null) sp.set('offset', String(params.offset))
  const s = sp.toString()
  return s ? `?${s}` : ''
}

export const ticketsApi = {
  list: (token: string | undefined, params: ListParams = {}) =>
    api<{ items: WorkItem[] }>(token, `/v1/tickets${query(params)}`),

  get: (token: string | undefined, id: string) =>
    api<WorkItem>(token, `/v1/tickets/${id}`),

  create: (token: string | undefined, body: CreateWorkItem) =>
    api<WorkItem>(token, '/v1/tickets', { method: 'POST', body: JSON.stringify(body) }),

  update: (
    token: string | undefined,
    id: string,
    body: Partial<{ title: string; description: string; priority: Priority; assignee_id: string; assignment_group: string }>,
  ) => api<WorkItem>(token, `/v1/tickets/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),

  transition: (token: string | undefined, id: string, status: Status, comment?: string) =>
    api<WorkItem>(token, `/v1/tickets/${id}/transition`, {
      method: 'POST',
      body: JSON.stringify({ status, comment }),
    }),

  comments: (token: string | undefined, id: string) =>
    api<{ items: Comment[] }>(token, `/v1/tickets/${id}/comments`),

  addComment: (token: string | undefined, id: string, body: string, internal: boolean) =>
    api<Comment>(token, `/v1/tickets/${id}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body, internal }),
    }),
}
