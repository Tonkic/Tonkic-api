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
import { z } from 'zod'

export const createTicketSchema = z.object({
  title: z.string().trim().min(1).max(120),
  category: z.enum(['account', 'billing', 'api', 'bug', 'suggestion', 'other']),
  content: z.string().trim().min(1).max(5000),
})

export const replyTicketSchema = z.object({
  content: z.string().trim().min(1).max(5000),
})

export type CreateTicketFormValues = z.infer<typeof createTicketSchema>
export type ReplyTicketFormValues = z.infer<typeof replyTicketSchema>
