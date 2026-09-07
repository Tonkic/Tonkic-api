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
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useDebounce } from '@/hooks'

import { getTickets } from '../api'
import {
  TICKET_CATEGORY_LABEL_KEYS,
  TICKET_PRIORITY_OPTIONS,
  TICKET_STATUS_OPTIONS,
} from '../constants'
import { TicketPriorityBadge, TicketStatusBadge } from './ticket-status-badges'

type TicketsListProps = {
  admin: boolean
  onOpenTicket: (ticketId: number) => void
}

const PAGE_SIZE = 10
const TICKET_SKELETON_KEYS = [
  'ticket-skeleton-1',
  'ticket-skeleton-2',
  'ticket-skeleton-3',
  'ticket-skeleton-4',
  'ticket-skeleton-5',
] as const

export function TicketsList(props: TicketsListProps) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState(0)
  const [priority, setPriority] = useState(0)
  const [keyword, setKeyword] = useState('')
  const debouncedKeyword = useDebounce(keyword, 350)
  const query = useQuery({
    queryKey: [
      'tickets',
      props.admin,
      page,
      status,
      priority,
      debouncedKeyword,
    ],
    queryFn: () =>
      getTickets({
        admin: props.admin,
        page,
        pageSize: PAGE_SIZE,
        status,
        priority,
        keyword: debouncedKeyword,
      }),
    // An empty ticket list is a valid state. Keep a transient ticket API
    // failure local to this page instead of sending the whole app to /500.
    meta: { suppressGlobalError: true },
  })
  const totalPages = Math.max(
    1,
    Math.ceil((query.data?.total ?? 0) / PAGE_SIZE)
  )
  let content: ReactNode
  if (query.isLoading) {
    content = (
      <div className='space-y-2 p-4'>
        {TICKET_SKELETON_KEYS.map((key) => (
          <Skeleton key={key} className='h-12 w-full' />
        ))}
      </div>
    )
  } else if (query.isError) {
    content = (
      <div className='text-destructive py-12 text-center text-sm'>
        {query.error instanceof Error
          ? query.error.message
          : t('Failed to load tickets')}
      </div>
    )
  } else if ((query.data?.items ?? []).length === 0) {
    content = (
      <div className='text-muted-foreground py-12 text-center text-sm'>
        {t('No tickets found')}
      </div>
    )
  } else {
    content = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Ticket')}</TableHead>
            {props.admin && <TableHead>{t('Customer')}</TableHead>}
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Priority')}</TableHead>
            <TableHead>{t('Updated')}</TableHead>
            <TableHead className='w-20' />
          </TableRow>
        </TableHeader>
        <TableBody>
          {(query.data?.items ?? []).map((ticket) => (
            <TableRow key={ticket.id}>
              <TableCell className='max-w-80'>
                <div className='truncate font-medium'>{ticket.title}</div>
                <div className='text-muted-foreground text-xs'>
                  #{ticket.id} /{' '}
                  {t(TICKET_CATEGORY_LABEL_KEYS[ticket.category])}
                </div>
              </TableCell>
              {props.admin && (
                <TableCell>
                  {ticket.display_name || ticket.username || '-'}
                </TableCell>
              )}
              <TableCell>
                <TicketStatusBadge status={ticket.status} />
              </TableCell>
              <TableCell>
                <TicketPriorityBadge priority={ticket.priority} />
              </TableCell>
              <TableCell>
                {dayjs.unix(ticket.updated_time).format('MM-DD HH:mm')}
              </TableCell>
              <TableCell>
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  onClick={() => props.onOpenTicket(ticket.id)}
                >
                  {t('Open')}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center gap-2'>
        {props.admin && (
          <Input
            value={keyword}
            onChange={(event) => {
              setKeyword(event.target.value)
              setPage(1)
            }}
            placeholder={t('Search tickets')}
            className='w-full sm:w-64'
          />
        )}
        <Select
          items={[
            { value: '0', label: t('All statuses') },
            ...TICKET_STATUS_OPTIONS.map((item) => ({
              value: String(item.value),
              label: t(item.labelKey),
            })),
          ]}
          value={String(status)}
          onValueChange={(value) => {
            setStatus(Number(value))
            setPage(1)
          }}
        >
          <SelectTrigger className='w-40'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value='0'>{t('All statuses')}</SelectItem>
              {TICKET_STATUS_OPTIONS.map((item) => (
                <SelectItem key={item.value} value={String(item.value)}>
                  {t(item.labelKey)}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        {props.admin && (
          <Select
            items={[
              { value: '0', label: t('All priorities') },
              ...TICKET_PRIORITY_OPTIONS.map((item) => ({
                value: String(item.value),
                label: t(item.labelKey),
              })),
            ]}
            value={String(priority)}
            onValueChange={(value) => {
              setPriority(Number(value))
              setPage(1)
            }}
          >
            <SelectTrigger className='w-40'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                <SelectItem value='0'>{t('All priorities')}</SelectItem>
                {TICKET_PRIORITY_OPTIONS.map((item) => (
                  <SelectItem key={item.value} value={String(item.value)}>
                    {t(item.labelKey)}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        )}
      </div>

      <Card className='py-0'>
        <CardContent className='p-0'>{content}</CardContent>
      </Card>

      <div className='flex items-center justify-end gap-2'>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={page <= 1}
          onClick={() => setPage((value) => Math.max(1, value - 1))}
        >
          {t('Previous')}
        </Button>
        <span className='text-muted-foreground text-sm'>
          {t('Page {{page}} of {{pages}}', { page, pages: totalPages })}
        </span>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={page >= totalPages}
          onClick={() => setPage((value) => Math.min(totalPages, value + 1))}
        >
          {t('Next')}
        </Button>
      </div>
    </div>
  )
}
