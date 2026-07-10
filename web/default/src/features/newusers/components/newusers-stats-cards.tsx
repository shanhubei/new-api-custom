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
import { BarChart3, Users, Wallet } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatQuota } from '@/lib/format'

import type { NewuserListSummary } from '../types'

type NewusersStatsCardsProps = {
  summary?: NewuserListSummary
}

export function NewusersStatsCards({ summary }: NewusersStatsCardsProps) {
  const { t } = useTranslation()

  const cards = [
    {
      title: t('Total Third-Party Users'),
      value: summary?.total_users ?? 0,
      icon: Users,
    },
    {
      title: t('Active Users'),
      value: summary?.active_users ?? 0,
      icon: BarChart3,
    },
    {
      title: t('Total Used Quota'),
      value: formatQuota(summary?.total_used ?? 0),
      icon: Wallet,
    },
  ]

  return (
    <div className='grid gap-3 md:grid-cols-3'>
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader className='flex flex-row items-center justify-between pb-2'>
            <CardTitle className='text-sm font-medium'>{card.title}</CardTitle>
            <card.icon className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{card.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
