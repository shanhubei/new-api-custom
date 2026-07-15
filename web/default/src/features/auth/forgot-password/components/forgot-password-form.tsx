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
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowRight, CheckIcon, CopyIcon, Loader2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { z } from 'zod'

import { Turnstile } from '@/components/turnstile'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  resetPasswordBySms,
  sendPasswordResetEmail,
  sendPasswordResetSms,
} from '@/features/auth/api'
import {
  forgotPasswordFormSchema,
  forgotPasswordSmsFormSchema,
  PASSWORD_RESET_COUNTDOWN,
} from '@/features/auth/constants'
import { useTurnstile } from '@/features/auth/hooks/use-turnstile'
import { useStatus } from '@/hooks/use-status'
import { useCountdown } from '@/hooks/use-countdown'
import { copyToClipboard } from '@/lib/copy-to-clipboard'
import { cn } from '@/lib/utils'

export function ForgotPasswordForm({
  className,
  ...props
}: React.HTMLAttributes<HTMLFormElement>) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingSms, setIsSendingSms] = useState(false)
  const [resetMethod, setResetMethod] = useState<'email' | 'phone'>('email')
  const [newPassword, setNewPassword] = useState('')
  const [copied, setCopied] = useState(false)

  const smsResetEnabled = Boolean(
    status?.sms_login ??
      status?.data?.sms_login ??
      status?.sms_verification ??
      status?.data?.sms_verification
  )

  const {
    isTurnstileEnabled,
    turnstileSiteKey,
    turnstileToken,
    setTurnstileToken,
    validateTurnstile,
  } = useTurnstile()
  const {
    secondsLeft,
    isActive,
    start: startCountdown,
  } = useCountdown({ initialSeconds: PASSWORD_RESET_COUNTDOWN })
  const {
    secondsLeft: smsSecondsLeft,
    isActive: isSmsActive,
    start: startSmsCountdown,
  } = useCountdown({ initialSeconds: PASSWORD_RESET_COUNTDOWN })

  const emailForm = useForm<z.infer<typeof forgotPasswordFormSchema>>({
    resolver: zodResolver(forgotPasswordFormSchema),
    defaultValues: { email: '' },
  })
  const smsForm = useForm<z.infer<typeof forgotPasswordSmsFormSchema>>({
    resolver: zodResolver(forgotPasswordSmsFormSchema),
    defaultValues: { phone: '', verificationCode: '' },
  })
  const turnstileReady = !isTurnstileEnabled || Boolean(turnstileToken)

  const description = useMemo(() => {
    if (newPassword) {
      return t('auth.resetPasswordConfirm.success')
    }
    if (resetMethod === 'phone' && smsResetEnabled) {
      return t(
        'Enter your registered phone number and verification code to reset your password.'
      )
    }
    return t(
      'Enter your registered email and we will send you a link to reset your password.'
    )
  }, [newPassword, resetMethod, smsResetEnabled, t])

  async function onEmailSubmit(data: z.infer<typeof forgotPasswordFormSchema>) {
    if (!validateTurnstile()) return

    setIsLoading(true)
    try {
      const res = await sendPasswordResetEmail(data.email, turnstileToken)
      if (res?.success) {
        emailForm.reset()
        startCountdown()
        toast.success(t('Reset email sent, please check your inbox'))
      } else {
        toast.error(res?.message || t('Failed to send reset email'))
      }
    } catch (_error) {
      // Errors are handled by global interceptor
    } finally {
      setIsLoading(false)
    }
  }

  async function handleSendSmsCode() {
    if (!validateTurnstile()) return
    const phone = smsForm.getValues('phone')
    const valid = await smsForm.trigger('phone')
    if (!valid) return

    setIsSendingSms(true)
    try {
      const res = await sendPasswordResetSms(phone, turnstileToken)
      if (res?.success) {
        startSmsCountdown()
        toast.success(t('Verification code sent'))
      } else {
        toast.error(res?.message || t('Failed to send verification code'))
      }
    } catch (_error) {
      // Errors are handled by global interceptor
    } finally {
      setIsSendingSms(false)
    }
  }

  async function onSmsSubmit(data: z.infer<typeof forgotPasswordSmsFormSchema>) {
    if (!validateTurnstile()) return

    setIsLoading(true)
    try {
      const res = await resetPasswordBySms(
        data.phone,
        data.verificationCode,
        turnstileToken
      )
      if (res?.success && typeof res.data === 'string') {
        setNewPassword(res.data)
        const copySuccess = await copyToClipboard(res.data)
        if (copySuccess) {
          toast.success(
            t('Password reset and copied to clipboard: {{password}}', {
              password: res.data,
            })
          )
        } else {
          toast.success(t('Password reset: {{password}}', { password: res.data }))
        }
      } else {
        toast.error(res?.message || t('phone or verification code error'))
      }
    } catch (_error) {
      // Errors are handled by global interceptor
    } finally {
      setIsLoading(false)
    }
  }

  async function handleCopy() {
    if (!newPassword) return
    const copySuccess = await copyToClipboard(newPassword)
    if (copySuccess) {
      setCopied(true)
      toast.success(
        t('Password copied to clipboard: {{password}}', {
          password: newPassword,
        })
      )
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className={cn('grid gap-2', className)} {...props}>
      <p className='text-muted-foreground mb-2 text-left text-sm sm:text-base'>
        {description}
      </p>

      {smsResetEnabled && !newPassword && (
        <Tabs
          value={resetMethod}
          onValueChange={(value) =>
            setResetMethod(value === 'phone' ? 'phone' : 'email')
          }
          className='mb-2'
        >
          <TabsList className='grid w-full grid-cols-2'>
            <TabsTrigger value='email'>{t('Email')}</TabsTrigger>
            <TabsTrigger value='phone'>{t('Phone')}</TabsTrigger>
          </TabsList>
        </Tabs>
      )}

      {newPassword ? (
        <div className='space-y-3'>
          <FormItem>
            <FormLabel>{t('New password')}</FormLabel>
            <div className='flex gap-2'>
              <Input value={newPassword} disabled className='font-mono' />
              <Button type='button' variant='outline' onClick={handleCopy}>
                {copied ? <CheckIcon /> : <CopyIcon />}
              </Button>
            </div>
          </FormItem>
        </div>
      ) : resetMethod === 'phone' && smsResetEnabled ? (
        <Form {...smsForm}>
          <form
            onSubmit={smsForm.handleSubmit(onSmsSubmit)}
            className='grid gap-2'
          >
            <FormField
              control={smsForm.control}
              name='phone'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Phone')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('Enter your phone number')}
                      inputMode='numeric'
                      maxLength={11}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={smsForm.control}
              name='verificationCode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Verification code')}</FormLabel>
                  <div className='flex gap-2'>
                    <FormControl>
                      <Input
                        placeholder={t('Enter verification code')}
                        inputMode='numeric'
                        maxLength={6}
                        {...field}
                      />
                    </FormControl>
                    <Button
                      type='button'
                      variant='outline'
                      disabled={isSendingSms || isSmsActive || !turnstileReady}
                      onClick={handleSendSmsCode}
                    >
                      {isSmsActive
                        ? t('Resend ({{seconds}}s)', {
                            seconds: smsSecondsLeft,
                          })
                        : isSendingSms
                          ? t('Sending...')
                          : t('Send code')}
                    </Button>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              type='submit'
              className='mt-2'
              disabled={isLoading || !turnstileReady}
            >
              {t('Reset password')}
              {isLoading ? <Loader2 className='animate-spin' /> : <ArrowRight />}
            </Button>
          </form>
        </Form>
      ) : (
        <Form {...emailForm}>
          <form
            onSubmit={emailForm.handleSubmit(onEmailSubmit)}
            className='grid gap-2'
          >
            <FormField
              control={emailForm.control}
              name='email'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input placeholder='name@example.com' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              type='submit'
              className='mt-2'
              disabled={isLoading || isActive || !turnstileReady}
            >
              {isActive
                ? t('Resend ({{seconds}}s)', { seconds: secondsLeft })
                : t('Send reset email')}
              {isLoading ? <Loader2 className='animate-spin' /> : <ArrowRight />}
            </Button>
          </form>
        </Form>
      )}

      {isTurnstileEnabled && (
        <div className='mt-2'>
          <Turnstile siteKey={turnstileSiteKey} onVerify={setTurnstileToken} />
        </div>
      )}
    </div>
  )
}
