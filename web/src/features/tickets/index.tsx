/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { CreateTicketDialog } from './components/create-ticket-dialog'
import { TicketDetailDialog } from './components/ticket-detail-dialog'
import { TicketsList } from './components/tickets-list'

export function Tickets() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.USER)
  const admin = role >= ROLE.ADMIN
  const [createOpen, setCreateOpen] = useState(false)
  const [selectedTicketId, setSelectedTicketId] = useState<number | null>(null)

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Support tickets')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        {!admin && (
          <Button type='button' onClick={() => setCreateOpen(true)}>
            <Plus aria-hidden='true' />
            {t('Create ticket')}
          </Button>
        )}
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <TicketsList admin={admin} onOpenTicket={setSelectedTicketId} />
        <CreateTicketDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          onCreated={async (ticketId) => {
            await queryClient.invalidateQueries({ queryKey: ['tickets'] })
            setSelectedTicketId(ticketId)
          }}
        />
        <TicketDetailDialog
          ticketId={selectedTicketId}
          admin={admin}
          onOpenChange={(open) => {
            if (!open) setSelectedTicketId(null)
          }}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
