import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from 'react-oidc-context'
import { ticketsApi } from '../../api/tickets'
import type { CreateWorkItem, ListParams, Status } from '../../api/types'

function useToken(): string | undefined {
  return useAuth().user?.access_token
}

export function useTickets(params: ListParams) {
  const token = useToken()
  return useQuery({
    queryKey: ['tickets', params],
    queryFn: () => ticketsApi.list(token, params),
  })
}

export function useTicket(id: string) {
  const token = useToken()
  return useQuery({
    queryKey: ['ticket', id],
    queryFn: () => ticketsApi.get(token, id),
    enabled: !!id,
  })
}

export function useComments(id: string) {
  const token = useToken()
  return useQuery({
    queryKey: ['comments', id],
    queryFn: () => ticketsApi.comments(token, id),
    enabled: !!id,
  })
}

export function useCreateTicket() {
  const token = useToken()
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: CreateWorkItem) => ticketsApi.create(token, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tickets'] }),
  })
}

export function useUpdateTicket(id: string) {
  const token = useToken()
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Partial<{ assignee_id: string; assignment_group: string }>) =>
      ticketsApi.update(token, id, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ticket', id] })
      void qc.invalidateQueries({ queryKey: ['tickets'] })
    },
  })
}

export function useTransition(id: string) {
  const token = useToken()
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars: { status: Status; comment?: string }) =>
      ticketsApi.transition(token, id, vars.status, vars.comment),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ticket', id] })
      void qc.invalidateQueries({ queryKey: ['comments', id] })
      void qc.invalidateQueries({ queryKey: ['tickets'] })
    },
  })
}

export function useAddComment(id: string) {
  const token = useToken()
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars: { body: string; internal: boolean }) =>
      ticketsApi.addComment(token, id, vars.body, vars.internal),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['comments', id] }),
  })
}
