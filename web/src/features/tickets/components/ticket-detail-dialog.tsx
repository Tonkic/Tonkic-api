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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

import { closeTicket, getTicket, replyTicket, updateTicket } from '../api'
import {
  TICKET_CATEGORY_LABEL_KEYS,
  TICKET_PRIORITY_OPTIONS,
  TICKET_STATUS_OPTIONS,
} from '../constants'
import { replyTicketSchema, type ReplyTicketFormValues } from '../lib/schemas'
import { TICKET_STATUS } from '../types'
import { TicketPriorityBadge, TicketStatusBadge } from './ticket-status-badges'

type TicketDetailDialogProps = {
  ticketId: number | null
  admin: boolean
  onOpenChange: (open: boolean) => void
}

export function TicketDetailDialog(props: TicketDetailDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const form = useForm<ReplyTicketFormValues>({
    resolver: zodResolver(replyTicketSchema),
    defaultValues: { content: '' },
  })
  const query = useQuery({
    queryKey: ['ticket', props.ticketId, props.admin],
    queryFn: () => getTicket(props.ticketId as number, props.admin),
    enabled: props.ticketId != null,
    // Render the dialog's local error state for ticket-specific failures.
    meta: { suppressGlobalError: true },
  })

  useEffect(() => {
    form.reset()
  }, [form, props.ticketId])

  const refreshTickets = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['tickets'] }),
      queryClient.invalidateQueries({
        queryKey: ['ticket', props.ticketId, props.admin],
      }),
    ])
  }
  const replyMutation = useMutation({
    mutationFn: (values: ReplyTicketFormValues) =>
      replyTicket(props.ticketId as number, values.content, props.admin),
    onSuccess: async () => {
      form.reset()
      toast.success(t('Reply sent'))
      await refreshTickets()
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t('Request failed')),
  })
  const closeMutation = useMutation({
    mutationFn: () => closeTicket(props.ticketId as number),
    onSuccess: async () => {
      toast.success(t('Ticket closed'))
      await refreshTickets()
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t('Request failed')),
  })
  const updateMutation = useMutation({
    mutationFn: (payload: { status?: number; priority?: number }) =>
      updateTicket(props.ticketId as number, payload),
    onSuccess: async () => {
      toast.success(t('Ticket updated'))
      await refreshTickets()
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t('Request failed')),
  })

  const ticket = query.data
  const isClosed = ticket?.status === TICKET_STATUS.CLOSED

  return (
    <Dialog
      open={props.ticketId != null}
      onOpenChange={props.onOpenChange}
      title={ticket?.title ?? t('Ticket details')}
      description={
        ticket
          ? `${t('Ticket')} #${ticket.id} · ${dayjs.unix(ticket.created_time).format('YYYY-MM-DD HH:mm')}`
          : t('Loading...')
      }
      contentClassName='sm:max-w-3xl'
      contentHeight='min(70vh, 760px)'
    >
      {query.isLoading && (
        <div className='space-y-3'>
          <Skeleton className='h-10 w-full' />
          <Skeleton className='h-28 w-full' />
          <Skeleton className='h-20 w-4/5' />
        </div>
      )}
      {!query.isLoading && (query.isError || !ticket) && (
        <div className='text-destructive py-8 text-center text-sm'>
          {query.error instanceof Error
            ? query.error.message
            : t('Failed to load ticket')}
        </div>
      )}
      {!query.isLoading && !query.isError && ticket && (
        <div className='space-y-4'>
          <div className='bg-muted/40 flex flex-wrap items-center gap-2 rounded-lg p-3'>
            <TicketStatusBadge status={ticket.status} />
            <TicketPriorityBadge priority={ticket.priority} />
            <span className='text-muted-foreground text-sm'>
              {t('Category')}: {t(TICKET_CATEGORY_LABEL_KEYS[ticket.category])}
            </span>
            {props.admin && (
              <span className='text-muted-foreground text-sm'>
                {t('Customer')}: {ticket.display_name || ticket.username}
              </span>
            )}
          </div>

          {props.admin && (
            <div className='grid gap-3 sm:grid-cols-2'>
              <Select
                items={TICKET_STATUS_OPTIONS.map((item) => ({
                  value: String(item.value),
                  label: t(item.labelKey),
                }))}
                value={String(ticket.status)}
                onValueChange={(value) =>
                  updateMutation.mutate({ status: Number(value) })
                }
                disabled={updateMutation.isPending}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {TICKET_STATUS_OPTIONS.map((item) => (
                      <SelectItem key={item.value} value={String(item.value)}>
                        {t(item.labelKey)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <Select
                items={TICKET_PRIORITY_OPTIONS.map((item) => ({
                  value: String(item.value),
                  label: t(item.labelKey),
                }))}
                value={String(ticket.priority)}
                onValueChange={(value) =>
                  updateMutation.mutate({ priority: Number(value) })
                }
                disabled={updateMutation.isPending}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {TICKET_PRIORITY_OPTIONS.map((item) => (
                      <SelectItem key={item.value} value={String(item.value)}>
                        {t(item.labelKey)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          )}

          <div className='space-y-3' aria-label={t('Conversation')}>
            {(ticket.messages ?? []).map((message) => (
              <div
                key={message.id}
                className={cn(
                  'max-w-[88%] rounded-xl px-3 py-2.5',
                  message.is_admin
                    ? 'bg-primary/10 mr-auto'
                    : 'bg-muted ml-auto'
                )}
              >
                <div className='mb-1 flex items-center justify-between gap-3 text-xs'>
                  <span className='font-medium'>
                    {message.is_admin ? t('Support') : t('Customer')}
                  </span>
                  <span className='text-muted-foreground'>
                    {dayjs.unix(message.created_time).format('MM-DD HH:mm')}
                  </span>
                </div>
                <p className='text-sm break-words whitespace-pre-wrap'>
                  {message.content}
                </p>
              </div>
            ))}
          </div>

          {!isClosed && (
            <Form {...form}>
              <form
                className='space-y-2 border-t pt-4'
                onSubmit={form.handleSubmit((values) =>
                  replyMutation.mutate(values)
                )}
              >
                <FormField
                  control={form.control}
                  name='content'
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <Textarea
                          {...field}
                          maxLength={5000}
                          className='min-h-24'
                          placeholder={t('Write a reply...')}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className='flex justify-between gap-2'>
                  {!props.admin && (
                    <Button
                      type='button'
                      variant='outline'
                      disabled={closeMutation.isPending}
                      onClick={() => closeMutation.mutate()}
                    >
                      {t('Close ticket')}
                    </Button>
                  )}
                  <Button
                    type='submit'
                    className='ml-auto'
                    disabled={replyMutation.isPending}
                  >
                    {replyMutation.isPending
                      ? t('Sending...')
                      : t('Send reply')}
                  </Button>
                </div>
              </form>
            </Form>
          )}
        </div>
      )}
    </Dialog>
  )
}
