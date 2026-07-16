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
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { PasswordInput } from '@/components/password-input'
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
import { api } from '@/lib/api'

type NewuserRegisterFormProps = {
  initialCode: string
}

type RegisterInfo = {
  owner_user_id: number
  owner_username: string
  owner_display_name: string
  register_code: string
}

export function NewuserRegisterForm({ initialCode }: NewuserRegisterFormProps) {
  const { t } = useTranslation()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const formSchema = z.object({
    register_code: z.string().min(1, t('Invite code is required')),
    username: z
      .string()
      .min(1, t('Username is required'))
      .max(64, t('Username is too long')),
    password: z
      .string()
      .min(8, t('Password must be at least 8 characters'))
      .max(64, t('Password is too long')),
    display_name: z.string().max(64).optional(),
    email: z
      .string()
      .email(t('Please enter a valid email address'))
      .optional()
      .or(z.literal('')),
    phone: z.string().max(20).optional(),
  })

  type FormValues = z.infer<typeof formSchema>

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      register_code: initialCode,
      username: '',
      password: '',
      display_name: '',
      email: '',
      phone: '',
    },
  })

  useEffect(() => {
    if (initialCode) {
      form.setValue('register_code', initialCode)
    }
  }, [initialCode, form])

  const registerCode = form.watch('register_code')

  const infoQuery = useQuery({
    queryKey: ['newuser-register-info', registerCode],
    enabled: Boolean(registerCode?.trim()),
    queryFn: async () => {
      const res = await api.get('/api/newuser/register/info', {
        params: { code: registerCode.trim() },
      })
      if (!res.data?.success) {
        throw new Error(res.data?.message || t('Invalid invite code'))
      }
      return res.data.data as RegisterInfo
    },
    retry: false,
  })

  const onSubmit = async (values: FormValues) => {
    setIsSubmitting(true)
    try {
      const res = await api.post('/api/newuser/register', {
        register_type: 'member',
        register_code: values.register_code.trim(),
        username: values.username.trim(),
        password: values.password,
        display_name: values.display_name?.trim() || '',
        email: values.email?.trim() || '',
        phone: values.phone?.trim() || '',
      })
      if (!res.data?.success) {
        toast.error(res.data?.message || t('Registration failed'))
        return
      }
      toast.success(
        t(
          'Team registration successful. Please sign in on the desktop client.'
        )
      )
      form.reset({
        register_code: values.register_code,
        username: '',
        password: '',
        display_name: '',
        email: '',
        phone: '',
      })
    } catch {
      toast.error(t('Registration failed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-4'>
        <FormField
          control={form.control}
          name='register_code'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Invite Code')}</FormLabel>
              <FormControl>
                <Input placeholder={t('Enter invite code')} {...field} />
              </FormControl>
              {infoQuery.data ? (
                <FormDescription>
                  {t('Joining team')}:{' '}
                  {infoQuery.data.owner_display_name ||
                    infoQuery.data.owner_username}
                </FormDescription>
              ) : null}
              {infoQuery.isError ? (
                <p className='text-destructive text-sm'>
                  {t('Invalid invite code')}
                </p>
              ) : null}
              <FormMessage />
            </FormItem>
          )}
        />

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

        <FormField
          control={form.control}
          name='password'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Password')}</FormLabel>
              <FormControl>
                <PasswordInput
                  placeholder={t('Enter password (8-64 characters)')}
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
          name='email'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Email')}</FormLabel>
              <FormControl>
                <Input
                  type='email'
                  placeholder={t('Optional email address')}
                  {...field}
                />
              </FormControl>
              <FormDescription>
                {t(
                  'Optional. If email or phone is filled, the user can reset their password themselves.'
                )}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name='phone'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Phone')}</FormLabel>
              <FormControl>
                <Input placeholder={t('Optional phone number')} {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button
          type='submit'
          className='w-full'
          disabled={isSubmitting || infoQuery.isError}
        >
          {isSubmitting ? (
            <>
              <Loader2 className='mr-2 size-4 animate-spin' />
              {t('Registering...')}
            </>
          ) : (
            t('Register')
          )}
        </Button>
      </form>
    </Form>
  )
}
