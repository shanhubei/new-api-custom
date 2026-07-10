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
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { getCurrencyLabel } from '@/lib/currency'

import { createNewuser, updateNewuser } from '../api'
import { ERROR_MESSAGES, NEWUSER_STATUS, SUCCESS_MESSAGES } from '../constants'
import type { NewuserItem } from '../types'
import { useNewusers } from './newusers-provider'

type NewusersMutateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: NewuserItem
}

function getFormSchema(isUpdate: boolean, t: (key: string) => string) {
  return z.object({
    username: isUpdate
      ? z.string().optional()
      : z
          .string()
          .min(1, t('Username is required'))
          .max(64, t('Username is too long')),
    password: isUpdate
      ? z.string().optional()
      : z
          .string()
          .min(8, t('Password must be at least 8 characters'))
          .max(64, t('Password is too long')),
    display_name: z.string().max(64).optional(),
    quota_limit: z.coerce.number().min(0).optional(),
    status: z.coerce.number().optional(),
  })
}

export function NewusersMutateDrawer({
  open,
  onOpenChange,
  currentRow,
}: NewusersMutateDrawerProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh } = useNewusers()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const currencyLabel = getCurrencyLabel()

  const formSchema = getFormSchema(isUpdate, t)
  type FormValues = z.infer<typeof formSchema>

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: '',
      password: '',
      display_name: '',
      quota_limit: 0,
      status: NEWUSER_STATUS.ENABLED,
    },
  })

  useEffect(() => {
    if (!open) return
    if (currentRow) {
      form.reset({
        username: currentRow.username,
        password: '',
        display_name: currentRow.display_name,
        quota_limit: currentRow.quota_limit,
        status: currentRow.status,
      })
      return
    }
    form.reset({
      username: '',
      password: '',
      display_name: '',
      quota_limit: 0,
      status: NEWUSER_STATUS.ENABLED,
    })
  }, [open, currentRow, form])

  const onSubmit = async (values: FormValues) => {
    setIsSubmitting(true)
    try {
      if (isUpdate && currentRow) {
        const payload = {
          display_name: values.display_name?.trim() || currentRow.display_name,
          status: values.status ?? currentRow.status,
          quota_limit: values.quota_limit ?? currentRow.quota_limit,
          password: values.password?.trim() || undefined,
        }
        const result = await updateNewuser(currentRow.id, payload)
        if (result.success) {
          toast.success(t(SUCCESS_MESSAGES.UPDATED))
          onOpenChange(false)
          triggerRefresh()
        } else {
          toast.error(result.message || t(ERROR_MESSAGES.UNEXPECTED))
        }
        return
      }

      const result = await createNewuser({
        username: values.username?.trim() || '',
        password: values.password || '',
        display_name: values.display_name?.trim() || values.username?.trim() || '',
        quota_limit: values.quota_limit ?? 0,
      })
      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.CREATED))
        onOpenChange(false)
        triggerRefresh()
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.UNEXPECTED))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={sideDrawerContentClassName}>
        <SheetHeader className={sideDrawerHeaderClassName}>
          <SheetTitle>
            {isUpdate
              ? t('Update Third-Party User')
              : t('Create Third-Party User')}
          </SheetTitle>
          <SheetDescription>
            {isUpdate
              ? t('Update third-party user settings and quota limits.')
              : t(
                  'Create a third-party UI user mapped to an API key under your account.'
                )}
          </SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className={sideDrawerFormClassName}
          >
            {!isUpdate && (
              <FormField
                control={form.control}
                name='username'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Username')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('Enter username')} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <FormField
              control={form.control}
              name='password'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {isUpdate ? t('New Password') : t('Password')}
                  </FormLabel>
                  <FormControl>
                    <Input
                      type='password'
                      placeholder={
                        isUpdate
                          ? t('Leave blank to keep current password')
                          : t('Enter password (8-64 characters)')
                      }
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='display_name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Display Name')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('Leave blank to use username')}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='quota_limit'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('Quota Limit')} ({currencyLabel})
                  </FormLabel>
                  <FormControl>
                    <Input type='number' min={0} {...field} />
                  </FormControl>
                  <FormDescription>
                    {t('Set to 0 to share your account wallet without a per-user cap.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            {isUpdate && (
              <FormField
                control={form.control}
                name='status'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Status')}</FormLabel>
                    <Select
                      value={String(field.value ?? NEWUSER_STATUS.ENABLED)}
                      onValueChange={(value) => field.onChange(Number(value))}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value={String(NEWUSER_STATUS.ENABLED)}>
                          {t('Enabled')}
                        </SelectItem>
                        <SelectItem value={String(NEWUSER_STATUS.DISABLED)}>
                          {t('Disabled')}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <SheetFooter className={sideDrawerFooterClassName}>
              <SheetClose asChild>
                <Button type='button' variant='outline' disabled={isSubmitting}>
                  {t('Cancel')}
                </Button>
              </SheetClose>
              <Button type='submit' disabled={isSubmitting}>
                {isSubmitting ? t('Saving...') : t('Save')}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
