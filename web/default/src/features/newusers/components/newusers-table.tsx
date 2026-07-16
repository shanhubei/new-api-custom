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
import { Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatQuota, formatTimestamp } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getNewusers } from '../api'
import { ERROR_MESSAGES, NEWUSER_STATUS } from '../constants'
import type { NewuserItem } from '../types'
import { DataTableRowActions } from './data-table-row-actions'
import { useNewusers } from './newusers-provider'

function isDisabledRow(user: NewuserItem) {
  return user.status !== NEWUSER_STATUS.ENABLED
}

export function NewusersTable() {
  const { t } = useTranslation()
  const { refreshTrigger } = useNewusers()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['newusers', refreshTrigger],
    queryFn: async () => {
      const res = await getNewusers()
      if (!res.success) {
        throw new Error(res.message || t(ERROR_MESSAGES.UNEXPECTED))
      }
      return res.data
    },
    retry: false,
  })

  if (isError) {
    const message =
      error instanceof Error ? error.message : t(ERROR_MESSAGES.UNEXPECTED)
    const moduleDisabled =
      message.toLowerCase().includes('disabled') ||
      message.includes('503')

    return (
      <Alert variant={moduleDisabled ? 'destructive' : 'default'}>
        <AlertTitle>
          {moduleDisabled
            ? t('Team user module unavailable')
            : t('Failed to load team users')}
        </AlertTitle>
        <AlertDescription>
          {moduleDisabled
            ? t(ERROR_MESSAGES.MODULE_DISABLED)
            : message}
        </AlertDescription>
      </Alert>
    )
  }

  if (isLoading) {
    return (
      <div className='space-y-2'>
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className='h-12 w-full' />
        ))}
      </div>
    )
  }

  const items = data?.items ?? []

  if (!items.length) {
    return (
      <div className='rounded-lg border p-8'>
        <Empty className='border-none p-0'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Users className='size-6' />
            </EmptyMedia>
            <EmptyTitle>{t('No Team Users Yet')}</EmptyTitle>
            <EmptyDescription>
              {t(
                'Create a team user to issue dedicated API access for your external UI.'
              )}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className='rounded-lg border'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Username')}</TableHead>
            <TableHead>{t('Display Name')}</TableHead>
            <TableHead>{t('Email')}</TableHead>
            <TableHead>{t('Phone')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Used Quota')}</TableHead>
            <TableHead>{t('Quota Limit')}</TableHead>
            <TableHead>{t('Last Login')}</TableHead>
            <TableHead className='w-[70px]' />
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((user) => (
            <TableRow
              key={user.id}
              className={cn(isDisabledRow(user) && 'opacity-60')}
            >
              <TableCell className='font-medium'>{user.username}</TableCell>
              <TableCell>{user.display_name || user.username}</TableCell>
              <TableCell className='text-muted-foreground text-sm'>
                {user.email || '—'}
              </TableCell>
              <TableCell className='text-muted-foreground text-sm'>
                {user.phone || '—'}
              </TableCell>
              <TableCell>
                <StatusBadge
                  label={
                    user.status === NEWUSER_STATUS.ENABLED
                      ? t('Enabled')
                      : t('Disabled')
                  }
                  variant={
                    user.status === NEWUSER_STATUS.ENABLED
                      ? 'success'
                      : 'secondary'
                  }
                />
              </TableCell>
              <TableCell>{formatQuota(user.used_quota)}</TableCell>
              <TableCell>
                {user.quota_limit > 0
                  ? formatQuota(user.quota_limit)
                  : t('Unlimited')}
              </TableCell>
              <TableCell className='text-muted-foreground text-xs'>
                {user.last_login_time
                  ? formatTimestamp(user.last_login_time)
                  : t('Never')}
              </TableCell>
              <TableCell>
                <DataTableRowActions row={user} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
