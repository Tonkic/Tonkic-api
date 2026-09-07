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
import type { StatusVariant } from '@/components/status-badge'

import { TICKET_PRIORITY, TICKET_STATUS, type TicketCategory } from './types'

export const TICKET_CATEGORY_LABEL_KEYS: Record<TicketCategory, string> = {
  account: 'Account',
  billing: 'Billing',
  api: 'API',
  bug: 'Bug',
  suggestion: 'Suggestion',
  other: 'Other',
}

export const TICKET_CATEGORIES = [
  { value: 'account', labelKey: TICKET_CATEGORY_LABEL_KEYS.account },
  { value: 'billing', labelKey: TICKET_CATEGORY_LABEL_KEYS.billing },
  { value: 'api', labelKey: TICKET_CATEGORY_LABEL_KEYS.api },
  { value: 'bug', labelKey: TICKET_CATEGORY_LABEL_KEYS.bug },
  { value: 'suggestion', labelKey: TICKET_CATEGORY_LABEL_KEYS.suggestion },
  { value: 'other', labelKey: TICKET_CATEGORY_LABEL_KEYS.other },
] as const

export const TICKET_STATUS_OPTIONS = [
  {
    value: TICKET_STATUS.PENDING,
    labelKey: 'Pending support',
    variant: 'warning' as StatusVariant,
  },
  {
    value: TICKET_STATUS.ANSWERED,
    labelKey: 'Waiting for you',
    variant: 'info' as StatusVariant,
  },
  {
    value: TICKET_STATUS.CLOSED,
    labelKey: 'Closed',
    variant: 'neutral' as StatusVariant,
  },
] as const

export const TICKET_PRIORITY_OPTIONS = [
  {
    value: TICKET_PRIORITY.LOW,
    labelKey: 'Low',
    variant: 'neutral' as StatusVariant,
  },
  {
    value: TICKET_PRIORITY.NORMAL,
    labelKey: 'Normal',
    variant: 'info' as StatusVariant,
  },
  {
    value: TICKET_PRIORITY.HIGH,
    labelKey: 'High',
    variant: 'danger' as StatusVariant,
  },
] as const
