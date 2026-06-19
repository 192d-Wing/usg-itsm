// Mirrors the ticketing service JSON (services/ticketing/internal/domain).

export type WorkItemType = 'incident' | 'service_request'

export type Status =
  | 'new'
  | 'in_progress'
  | 'on_hold'
  | 'resolved'
  | 'closed'
  | 'cancelled'

export type Priority = 'critical' | 'high' | 'moderate' | 'low'

export interface WorkItem {
  id: string
  number: string
  type: WorkItemType
  title: string
  description: string
  status: Status
  priority: Priority
  requester_id: string
  assignee_id?: string | null
  assignment_group?: string | null
  created_at: string
  updated_at: string
  closed_at?: string | null
}

export interface Comment {
  id: string
  work_item_id: string
  author_id: string
  body: string
  internal: boolean
  created_at: string
}

export interface CreateWorkItem {
  type: WorkItemType
  title: string
  description?: string
  priority: Priority
  assignment_group?: string
}

export interface ListParams {
  type?: WorkItemType
  status?: Status
  assignee?: string
  limit?: number
  offset?: number
}
