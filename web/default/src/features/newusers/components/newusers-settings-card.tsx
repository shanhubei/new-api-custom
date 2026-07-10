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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { getNewuserSettings, updateNewuserSettings } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'

type NewusersSettingsCardProps = {
  refreshTrigger: number
}

export function NewusersSettingsCard({
  refreshTrigger,
}: NewusersSettingsCardProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [enabled, setEnabled] = useState(false)
  const [registerEnabled, setRegisterEnabled] = useState(false)
  const [registerCode, setRegisterCode] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['newuser-settings', refreshTrigger],
    queryFn: async () => {
      const res = await getNewuserSettings()
      if (!res.success) {
        throw new Error(res.message || t(ERROR_MESSAGES.UNEXPECTED))
      }
      return res.data
    },
  })

  useEffect(() => {
    if (!data) return
    setEnabled(data.enabled)
    setRegisterEnabled(data.register_enabled)
    setRegisterCode(data.register_code)
  }, [data])

  const mutation = useMutation({
    mutationFn: updateNewuserSettings,
    onSuccess: (res) => {
      if (res.success) {
        toast.success(t(SUCCESS_MESSAGES.SETTINGS_SAVED))
        queryClient.invalidateQueries({ queryKey: ['newuser-settings'] })
        queryClient.invalidateQueries({ queryKey: ['newusers'] })
      } else {
        toast.error(res.message || t(ERROR_MESSAGES.UNEXPECTED))
      }
    },
    onError: () => {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    },
  })

  const handleSave = () => {
    if (registerEnabled && !registerCode.trim()) {
      toast.error(t('Registration invite code is required when self-registration is enabled'))
      return
    }
    mutation.mutate({
      enabled,
      register_enabled: enabled ? registerEnabled : false,
      register_code: registerCode.trim(),
    })
  }

  if (isLoading) {
    return null
  }

  return (
    <Card>
      <CardHeader className='pb-3'>
        <CardTitle className='text-base'>
          {t('Third-Party User Settings')}
        </CardTitle>
        <CardDescription>
          {t(
            'Enable third-party users for your organization and configure self-registration.'
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='flex items-center justify-between gap-4'>
          <div>
            <Label htmlFor='newuser-enabled'>{t('Enable Third-Party Users')}</Label>
            <p className='text-muted-foreground text-xs'>
              {t('Allow creating and managing third-party UI users under your account.')}
            </p>
          </div>
          <Switch
            id='newuser-enabled'
            checked={enabled}
            onCheckedChange={(checked) => {
              setEnabled(checked)
              if (!checked) {
                setRegisterEnabled(false)
              }
            }}
          />
        </div>

        <div className='flex items-center justify-between gap-4'>
          <div>
            <Label htmlFor='newuser-register'>{t('Allow Self-Registration')}</Label>
            <p className='text-muted-foreground text-xs'>
              {t('Third-party UI can register via /api/newuser/register with an invite code.')}
            </p>
          </div>
          <Switch
            id='newuser-register'
            checked={registerEnabled}
            disabled={!enabled}
            onCheckedChange={setRegisterEnabled}
          />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='newuser-register-code'>{t('Registration Invite Code')}</Label>
          <Input
            id='newuser-register-code'
            value={registerCode}
            disabled={!enabled}
            onChange={(event) => setRegisterCode(event.target.value)}
            placeholder={t('Enter invite code')}
          />
        </div>

        <Button
          size='sm'
          onClick={handleSave}
          disabled={mutation.isPending}
        >
          {mutation.isPending ? t('Saving...') : t('Save Settings')}
        </Button>
      </CardContent>
    </Card>
  )
}
