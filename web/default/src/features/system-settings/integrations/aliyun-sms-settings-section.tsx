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
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import {
  SettingsForm,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const aliyunSmsSchema = z.object({
  AliyunSmsAccessKeyId: z.string(),
  AliyunSmsAccessKeySecret: z.string(),
  AliyunSmsSignName: z.string(),
  AliyunSmsTemplateCode: z.string(),
  AliyunSmsTemplateParamCodeKey: z.string(),
  AliyunSmsEndpoint: z.string(),
})

type AliyunSmsFormValues = z.infer<typeof aliyunSmsSchema>

type AliyunSmsSettingsSectionProps = {
  defaultValues: AliyunSmsFormValues
}

export function AliyunSmsSettingsSection({
  defaultValues,
}: AliyunSmsSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<AliyunSmsFormValues>({
    resolver: zodResolver(aliyunSmsSchema),
    defaultValues,
  })

  useResetForm(form, defaultValues)

  const onSubmit = async (values: AliyunSmsFormValues) => {
    const sanitized = {
      AliyunSmsAccessKeyId: values.AliyunSmsAccessKeyId.trim(),
      AliyunSmsAccessKeySecret: values.AliyunSmsAccessKeySecret.trim(),
      AliyunSmsSignName: values.AliyunSmsSignName.trim(),
      AliyunSmsTemplateCode: values.AliyunSmsTemplateCode.trim(),
      AliyunSmsTemplateParamCodeKey: values.AliyunSmsTemplateParamCodeKey.trim(),
      AliyunSmsEndpoint: values.AliyunSmsEndpoint.trim(),
    }

    const initial = {
      AliyunSmsAccessKeyId: defaultValues.AliyunSmsAccessKeyId.trim(),
      AliyunSmsAccessKeySecret: defaultValues.AliyunSmsAccessKeySecret.trim(),
      AliyunSmsSignName: defaultValues.AliyunSmsSignName.trim(),
      AliyunSmsTemplateCode: defaultValues.AliyunSmsTemplateCode.trim(),
      AliyunSmsTemplateParamCodeKey:
        defaultValues.AliyunSmsTemplateParamCodeKey.trim(),
      AliyunSmsEndpoint: defaultValues.AliyunSmsEndpoint.trim(),
    }

    const updates: Array<{ key: string; value: string }> = []

    if (sanitized.AliyunSmsAccessKeyId !== initial.AliyunSmsAccessKeyId) {
      updates.push({
        key: 'AliyunSmsAccessKeyId',
        value: sanitized.AliyunSmsAccessKeyId,
      })
    }

    if (
      sanitized.AliyunSmsAccessKeySecret &&
      sanitized.AliyunSmsAccessKeySecret !== initial.AliyunSmsAccessKeySecret
    ) {
      updates.push({
        key: 'AliyunSmsAccessKeySecret',
        value: sanitized.AliyunSmsAccessKeySecret,
      })
    }

    if (sanitized.AliyunSmsSignName !== initial.AliyunSmsSignName) {
      updates.push({
        key: 'AliyunSmsSignName',
        value: sanitized.AliyunSmsSignName,
      })
    }

    if (sanitized.AliyunSmsTemplateCode !== initial.AliyunSmsTemplateCode) {
      updates.push({
        key: 'AliyunSmsTemplateCode',
        value: sanitized.AliyunSmsTemplateCode,
      })
    }

    if (
      sanitized.AliyunSmsTemplateParamCodeKey !==
      initial.AliyunSmsTemplateParamCodeKey
    ) {
      updates.push({
        key: 'AliyunSmsTemplateParamCodeKey',
        value: sanitized.AliyunSmsTemplateParamCodeKey,
      })
    }

    if (sanitized.AliyunSmsEndpoint !== initial.AliyunSmsEndpoint) {
      updates.push({
        key: 'AliyunSmsEndpoint',
        value: sanitized.AliyunSmsEndpoint,
      })
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('Aliyun SMS')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save Aliyun SMS settings'
          />
          <FormField
            control={form.control}
            name='AliyunSmsAccessKeyId'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('AccessKey ID')}</FormLabel>
                <FormControl>
                  <Input
                    autoComplete='off'
                    placeholder={t('AccessKey ID')}
                    {...field}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t('Aliyun Cloud AccessKey ID for Dysmsapi')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='AliyunSmsAccessKeySecret'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('AccessKey Secret')}</FormLabel>
                <FormControl>
                  <Input
                    autoComplete='off'
                    type='password'
                    placeholder={t('Enter new secret to update')}
                    {...field}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t('Leave blank to keep the existing secret')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='grid gap-6 md:grid-cols-2'>
            <FormField
              control={form.control}
              name='AliyunSmsSignName'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Sign Name')}</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete='off'
                      placeholder={t('Sign Name')}
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('SMS signature name approved in Aliyun console')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='AliyunSmsTemplateCode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Template Code')}</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete='off'
                      placeholder={t('Template Code')}
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('SMS template CODE from Aliyun console')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className='grid gap-6 md:grid-cols-2'>
            <FormField
              control={form.control}
              name='AliyunSmsTemplateParamCodeKey'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Template Param Code Key')}</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete='off'
                      placeholder='code'
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'JSON key for the verification code variable in the template'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='AliyunSmsEndpoint'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Endpoint')}</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete='off'
                      placeholder={t('Optional Dysmsapi endpoint')}
                      {...field}
                      onChange={(event) => field.onChange(event.target.value)}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Leave blank to use the default Aliyun Dysmsapi endpoint')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
