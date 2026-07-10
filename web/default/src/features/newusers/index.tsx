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

import { SectionPageLayout } from '@/components/layout'

import { getNewuserSettings, getNewusers } from './api'
import { NewusersDeleteDialog } from './components/newusers-delete-dialog'
import { NewusersMutateDrawer } from './components/newusers-mutate-drawer'
import { NewusersPrimaryButtons } from './components/newusers-primary-buttons'
import { NewusersProvider, useNewusers } from './components/newusers-provider'
import { NewusersSettingsCard } from './components/newusers-settings-card'
import { NewusersStatsCards } from './components/newusers-stats-cards'
import { NewusersTable } from './components/newusers-table'
import { NewusersUsageDrawer } from './components/newusers-usage-drawer'

function NewusersStatsCardsWrapper() {
  const { refreshTrigger } = useNewusers()
  const { data } = useQuery({
    queryKey: ['newusers', refreshTrigger],
    queryFn: async () => {
      const res = await getNewusers()
      return res.success ? res.data : null
    },
    retry: false,
  })

  return <NewusersStatsCards summary={data?.summary} />
}

function NewusersContent() {
  const { t } = useTranslation()
  const { open, setOpen, currentRow, refreshTrigger } = useNewusers()

  const { data: settings } = useQuery({
    queryKey: ['newuser-settings', refreshTrigger],
    queryFn: async () => {
      const res = await getNewuserSettings()
      return res.success ? res.data : null
    },
    retry: false,
  })

  const orgEnabled = settings?.enabled ?? true

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Third-Party Users')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <NewusersPrimaryButtons disabled={!orgEnabled} />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content className='space-y-4'>
          <NewusersSettingsCard refreshTrigger={refreshTrigger} />
          <NewusersStatsCardsWrapper />
          <NewusersTable />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <NewusersMutateDrawer
        open={open === 'create' || open === 'update'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        currentRow={open === 'update' ? currentRow || undefined : undefined}
      />
      <NewusersUsageDrawer
        open={open === 'usage'}
        onOpenChange={(isOpen) => !isOpen && setOpen(null)}
        currentRow={currentRow}
      />
      <NewusersDeleteDialog />
    </>
  )
}

export function Newusers() {
  return (
    <NewusersProvider>
      <NewusersContent />
    </NewusersProvider>
  )
}
