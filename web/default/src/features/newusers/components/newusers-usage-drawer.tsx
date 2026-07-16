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
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Badge } from '@/components/ui/badge'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
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

import { getNewuserUsage } from '../api'
import type { NewuserItem } from '../types'

type NewusersUsageDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: NewuserItem | null
}

export function NewusersUsageDrawer({
  open,
  onOpenChange,
  currentRow,
}: NewusersUsageDrawerProps) {
  const { t } = useTranslation()

  const { data, isLoading } = useQuery({
    queryKey: ['newuser-usage', currentRow?.id],
    queryFn: async () => {
      if (!currentRow) return null
      const res = await getNewuserUsage(currentRow.id)
      if (!res.success) {
        throw new Error(res.message || t('Failed to load usage'))
      }
      return res.data ?? null
    },
    enabled: open && !!currentRow,
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={sideDrawerContentClassName}>
        <SheetHeader className={sideDrawerHeaderClassName}>
          <SheetTitle>{t('Team User Usage')}</SheetTitle>
          <SheetDescription>
            {currentRow
              ? t('Usage statistics for {{username}}', {
                  username: currentRow.username,
                })
              : t('Usage statistics')}
          </SheetDescription>
        </SheetHeader>

        {isLoading ? (
          <div className='space-y-3 px-4'>
            <Skeleton className='h-20 w-full' />
            <Skeleton className='h-40 w-full' />
          </div>
        ) : data ? (
          <div className='space-y-4 overflow-y-auto px-4 pb-6'>
            <div className='grid grid-cols-2 gap-3'>
              <div className='rounded-lg border p-3'>
                <div className='text-muted-foreground text-xs'>
                  {t('Used Quota')}
                </div>
                <div className='text-lg font-semibold'>
                  {formatQuota(data.used_quota)}
                </div>
                <div className='text-muted-foreground mt-1 text-[11px]'>
                  {t('This user')}
                </div>
              </div>
              <div className='rounded-lg border p-3'>
                <div className='text-muted-foreground text-xs'>
                  {t('Quota Limit')}
                </div>
                <div className='text-lg font-semibold'>
                  {data.quota_limit > 0
                    ? formatQuota(data.quota_limit)
                    : t('Unlimited')}
                </div>
                <div className='text-muted-foreground mt-1 text-[11px]'>
                  {t('Per-user cap')}
                </div>
              </div>
            </div>

            <div className='grid grid-cols-2 gap-3'>
              <div className='rounded-lg border p-3'>
                <div className='text-muted-foreground text-xs'>
                  {t('Organization Remaining Quota')}
                </div>
                <div className='text-lg font-semibold'>
                  {formatQuota(data.owner_quota ?? 0)}
                </div>
                <div className='text-muted-foreground mt-1 truncate text-[11px]'>
                  {data.owner_display_name || data.owner_username || t('Organization')}
                </div>
              </div>
              <div className='rounded-lg border p-3'>
                <div className='text-muted-foreground text-xs'>
                  {t('Organization Used Quota')}
                </div>
                <div className='text-lg font-semibold'>
                  {formatQuota(data.owner_used_quota ?? 0)}
                </div>
              </div>
            </div>

            <div className='flex flex-wrap gap-2'>
              <Badge variant='secondary'>
                {t('Token ID')}: {currentRow?.token_id}
              </Badge>
              {!data.unlimited && (
                <Badge variant='outline'>
                  {t('Remaining')}: {formatQuota(data.remain_quota)}
                </Badge>
              )}
            </div>

            <div>
              <h4 className='mb-2 text-sm font-medium'>
                {t('Recent Usage Logs')}
              </h4>
              {data.recent_logs?.length ? (
                <div className='rounded-lg border'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Time')}</TableHead>
                        <TableHead>{t('Model')}</TableHead>
                        <TableHead className='text-right'>
                          {t('Quota')}
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.recent_logs.slice(0, 20).map((log) => (
                        <TableRow key={log.id}>
                          <TableCell className='text-xs whitespace-nowrap'>
                            {formatTimestamp(log.created_at)}
                          </TableCell>
                          <TableCell className='max-w-[140px] truncate text-xs'>
                            {log.model_name || '-'}
                          </TableCell>
                          <TableCell className='text-right text-xs'>
                            {log.quota != null ? formatQuota(log.quota) : '-'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : (
                <div className='text-muted-foreground rounded-lg border p-6 text-center text-sm'>
                  {t('No usage logs yet')}
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  )
}
