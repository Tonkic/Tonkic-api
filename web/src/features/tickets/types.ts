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
export const TICKET_STATUS = {
  PENDING: 1,
  ANSWERED: 2,
  CLOSED: 3,
} as const

export const TICKET_PRIORITY = {
  LOW: 1,
  NORMAL: 2,
  HIGH: 3,
} as const

export type TicketCategory =
  | 'account'
  | 'billing'
  | 'api'
  | 'bug'
  | 'suggestion'
  | 'other'

export type TicketMessage = {
  id: number
  ticket_id: number
  user_id: number
  is_admin: boolean
  content: string
  created_time: number
}

export type Ticket = {
  id: number
  user_id: number
  title: string
  category: TicketCategory
  status: number
  priority: number
  created_time: number
  updated_time: number
  closed_time: number
  last_reply_time: number
  last_reply_by: number
  username?: string
  display_name?: string
}

export type TicketDetail = Ticket & {
  messages: TicketMessage[]
}

export type TicketPage = {
  page: number
  page_size: number
  total: number
  items: Ticket[]
}

export type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}

export type TicketListParams = {
  page: number
  pageSize: number
  status?: number
  priority?: number
  keyword?: string
  admin: boolean
}

export type CreateTicketPayload = {
  title: string
  category: TicketCategory
  content: string
}

export type AdminUpdateTicketPayload = {
  status?: number
  priority?: number
}
