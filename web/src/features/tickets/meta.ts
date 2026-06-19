import type { Priority, Status, WorkItemType } from '../../api/types'

export const STATUS_LABEL: Record<Status, string> = {
  new: 'New',
  in_progress: 'In Progress',
  on_hold: 'On Hold',
  resolved: 'Resolved',
  closed: 'Closed',
  cancelled: 'Cancelled',
}

export const STATUS_CLASS: Record<Status, string> = {
  new: 'bg-blue-100 text-blue-800',
  in_progress: 'bg-amber-100 text-amber-800',
  on_hold: 'bg-slate-200 text-slate-700',
  resolved: 'bg-emerald-100 text-emerald-800',
  closed: 'bg-slate-100 text-slate-500',
  cancelled: 'bg-rose-100 text-rose-700',
}

export const PRIORITY_LABEL: Record<Priority, string> = {
  critical: 'Critical',
  high: 'High',
  moderate: 'Moderate',
  low: 'Low',
}

export const PRIORITY_CLASS: Record<Priority, string> = {
  critical: 'bg-rose-100 text-rose-700',
  high: 'bg-orange-100 text-orange-700',
  moderate: 'bg-amber-100 text-amber-800',
  low: 'bg-slate-100 text-slate-600',
}

export const TYPE_LABEL: Record<WorkItemType, string> = {
  incident: 'Incident',
  service_request: 'Service Request',
}

// Allowed next states per the ticketing state machine (mirror of the Go domain).
export const TRANSITIONS: Record<Status, Status[]> = {
  new: ['in_progress', 'on_hold', 'resolved', 'cancelled'],
  in_progress: ['on_hold', 'resolved', 'cancelled'],
  on_hold: ['in_progress', 'resolved', 'cancelled'],
  resolved: ['in_progress', 'closed'],
  closed: [],
  cancelled: [],
}
