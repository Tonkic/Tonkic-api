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
import { api } from '@/lib/api'

import type {
  AdminUpdateTicketPayload,
  ApiResponse,
  CreateTicketPayload,
  TicketDetail,
  TicketListParams,
  TicketPage,
} from './types'

function unwrap<T>(response: ApiResponse<T>): T {
  if (!response.success) {
    throw new Error(response.message || 'Request failed')
  }
  return response.data
}

export async function getTickets(
  params: TicketListParams
): Promise<TicketPage> {
  const endpoint = params.admin ? '/api/ticket/admin' : '/api/ticket'
  const response = await api.get(endpoint, {
    params: {
      p: params.page,
      page_size: params.pageSize,
      status: params.status || undefined,
      priority: params.admin ? params.priority || undefined : undefined,
      keyword: params.admin ? params.keyword || undefined : undefined,
    },
  })
  const page = unwrap(response.data)
  return {
    ...page,
    // Older servers (and some database drivers) may encode empty slices as
    // null. Normalize at the API boundary so the UI can always treat this as
    // a collection.
    items: Array.isArray(page?.items) ? page.items : [],
  }
}

export async function getTicket(
  ticketId: number,
  admin: boolean
): Promise<TicketDetail> {
  const endpoint = admin
    ? `/api/ticket/admin/${ticketId}`
    : `/api/ticket/${ticketId}`
  const response = await api.get(endpoint)
  const ticket = unwrap(response.data)
  return {
    ...ticket,
    messages: Array.isArray(ticket?.messages) ? ticket.messages : [],
  }
}

export async function createTicket(
  payload: CreateTicketPayload
): Promise<TicketDetail> {
  const response = await api.post('/api/ticket', payload)
  return unwrap(response.data)
}

export async function replyTicket(
  ticketId: number,
  content: string,
  admin: boolean
): Promise<TicketDetail> {
  const endpoint = admin
    ? `/api/ticket/admin/${ticketId}/messages`
    : `/api/ticket/${ticketId}/messages`
  const response = await api.post(endpoint, { content })
  return unwrap(response.data)
}

export async function closeTicket(ticketId: number): Promise<TicketDetail> {
  const response = await api.patch(`/api/ticket/${ticketId}/close`)
  return unwrap(response.data)
}

export async function updateTicket(
  ticketId: number,
  payload: AdminUpdateTicketPayload
): Promise<TicketDetail> {
  const response = await api.patch(`/api/ticket/admin/${ticketId}`, payload)
  return unwrap(response.data)
}
