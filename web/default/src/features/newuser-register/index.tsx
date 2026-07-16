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
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { AuthLayout } from '@/features/auth/auth-layout'

import { NewuserRegisterForm } from './components/newuser-register-form'

type NewuserRegisterPageProps = {
  inviteCode: string
}

export function NewuserRegisterPage({ inviteCode }: NewuserRegisterPageProps) {
  const { t } = useTranslation()

  return (
    <AuthLayout>
      <div className='w-full space-y-8'>
        <div className='space-y-2'>
          <h2 className='text-center text-2xl font-semibold tracking-tight sm:text-left'>
            {t('Join Team')}
          </h2>
          <p className='text-muted-foreground text-left text-sm sm:text-base'>
            {t(
              'Register a team account with an invite code. This page is for registration only.'
            )}
          </p>
        </div>

        <Alert>
          <AlertTitle>{t('Desktop client login only')}</AlertTitle>
          <AlertDescription>
            {t(
              'Team accounts sign in on the desktop client only. Web sign-in is for organization admins, not team users.'
            )}
          </AlertDescription>
        </Alert>

        <NewuserRegisterForm initialCode={inviteCode} />
      </div>
    </AuthLayout>
  )
}
