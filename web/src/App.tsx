import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { RequireAuth } from './components/RequireAuth'
import { NewTicketPage } from './features/tickets/NewTicketPage'
import { TicketDetailPage } from './features/tickets/TicketDetailPage'
import { TicketsPage } from './features/tickets/TicketsPage'

export default function App() {
  return (
    <RequireAuth>
      <AppShell>
        <Routes>
          <Route path="/" element={<TicketsPage />} />
          <Route path="/new" element={<NewTicketPage />} />
          <Route path="/tickets/:id" element={<TicketDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AppShell>
    </RequireAuth>
  )
}
