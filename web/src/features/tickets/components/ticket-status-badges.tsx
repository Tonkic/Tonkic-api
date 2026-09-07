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
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'

import { TICKET_PRIORITY_OPTIONS, TICKET_STATUS_OPTIONS } from '../constants'

export function TicketStatusBadge(props: { status: number }) {
  const { t } = useTranslation()
  const option = TICKET_STATUS_OPTIONS.find(
    (item) => item.value === props.status
  )
  return (
    <StatusBadge
      label={t(option?.labelKey ?? 'Unknown')}
      variant={option?.variant ?? 'neutral'}
      copyable={false}
    />
  )
}

export function TicketPriorityBadge(props: { priority: number }) {
  const { t } = useTranslation()
  const option = TICKET_PRIORITY_OPTIONS.find(
    (item) => item.value === props.priority
  )
  return (
    <StatusBadge
      label={t(option?.labelKey ?? 'Unknown')}
      variant={option?.variant ?? 'neutral'}
      copyable={false}
    />
  )
}
