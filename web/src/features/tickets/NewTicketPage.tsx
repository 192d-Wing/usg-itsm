import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import type { Priority, WorkItemType } from '../../api/types'
import { Button } from '../../components/ui/Button'
import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Input, Label, Select, Textarea } from '../../components/ui/Field'
import { useCreateTicket } from './hooks'
import { PRIORITY_LABEL } from './meta'

export function NewTicketPage() {
  const navigate = useNavigate()
  const create = useCreateTicket()

  const [type, setType] = useState<WorkItemType>('incident')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<Priority>('moderate')

  function submit(e: React.FormEvent) {
    e.preventDefault()
    create.mutate(
      { type, title, description, priority },
      { onSuccess: (wi) => navigate(`/tickets/${wi.id}`) },
    )
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-4 text-xl font-semibold text-slate-800">New ticket</h1>
      <Card>
        <CardHeader title="Details" />
        <CardBody>
          <form className="space-y-4" onSubmit={submit}>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="type">Type</Label>
                <Select id="type" value={type} onChange={(e) => setType(e.target.value as WorkItemType)}>
                  <option value="incident">Incident</option>
                  <option value="service_request">Service Request</option>
                </Select>
              </div>
              <div>
                <Label htmlFor="priority">Priority</Label>
                <Select
                  id="priority"
                  value={priority}
                  onChange={(e) => setPriority(e.target.value as Priority)}
                >
                  {(Object.keys(PRIORITY_LABEL) as Priority[]).map((p) => (
                    <option key={p} value={p}>
                      {PRIORITY_LABEL[p]}
                    </option>
                  ))}
                </Select>
              </div>
            </div>
            <div>
              <Label htmlFor="title">Title</Label>
              <Input
                id="title"
                required
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Short summary"
              />
            </div>
            <div>
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What happened? What do you need?"
              />
            </div>

            {create.isError && (
              <p className="text-sm text-rose-700">
                {create.error instanceof ApiError ? create.error.message : 'Failed to create ticket.'}
              </p>
            )}

            <div className="flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => navigate('/')}>
                Cancel
              </Button>
              <Button type="submit" disabled={create.isPending || !title}>
                {create.isPending ? 'Creating…' : 'Create ticket'}
              </Button>
            </div>
          </form>
        </CardBody>
      </Card>
    </div>
  )
}
