/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useState } from 'react';
import {
  API,
  getLogo,
  showError,
  showInfo,
  showSuccess,
  getSystemName,
  copy,
} from '../../helpers';
import Turnstile from 'react-turnstile';
import { Button, Card, Form, TabPane, Tabs, Typography } from '@douyinfe/semi-ui';
import { IconMail, IconPhone } from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

const PasswordResetForm = () => {
  const { t } = useTranslation();
  const [inputs, setInputs] = useState({
    email: '',
    phone: '',
    verification_code: '',
  });
  const { email, phone, verification_code } = inputs;

  const [loading, setLoading] = useState(false);
  const [sendingSms, setSendingSms] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [disableButton, setDisableButton] = useState(false);
  const [countdown, setCountdown] = useState(30);
  const [smsResetEnabled, setSmsResetEnabled] = useState(false);
  const [activeTab, setActiveTab] = useState('email');
  const [newPassword, setNewPassword] = useState('');

  const logo = getLogo();
  const systemName = getSystemName();

  useEffect(() => {
    let status = localStorage.getItem('status');
    if (status) {
      status = JSON.parse(status);
      if (status.turnstile_check) {
        setTurnstileEnabled(true);
        setTurnstileSiteKey(status.turnstile_site_key);
      }
      setSmsResetEnabled(Boolean(status.sms_login || status.sms_verification));
    }
  }, []);

  useEffect(() => {
    let countdownInterval = null;
    if (disableButton && countdown > 0) {
      countdownInterval = setInterval(() => {
        setCountdown(countdown - 1);
      }, 1000);
    } else if (countdown === 0) {
      setDisableButton(false);
      setCountdown(30);
    }
    return () => clearInterval(countdownInterval);
  }, [disableButton, countdown]);

  function handleChange(name, value) {
    setInputs((prev) => ({ ...prev, [name]: value }));
  }

  async function handleEmailSubmit() {
    if (!email) {
      showError(t('请输入邮箱地址'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setDisableButton(true);
    setLoading(true);
    const res = await API.get(
      `/api/reset_password?email=${encodeURIComponent(email)}&turnstile=${turnstileToken}`,
    );
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('重置邮件发送成功，请检查邮箱！'));
      setInputs((prev) => ({ ...prev, email: '' }));
    } else {
      showError(message);
    }
    setLoading(false);
  }

  async function handleSendSmsCode() {
    if (!/^1\d{10}$/.test(phone)) {
      showError(t('请输入正确的手机号'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setSendingSms(true);
    setDisableButton(true);
    try {
      const res = await API.get(
        `/api/reset_password/sms?phone=${encodeURIComponent(phone)}&turnstile=${turnstileToken}`,
      );
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('验证码发送成功'));
      } else {
        showError(message);
        setDisableButton(false);
        setCountdown(30);
      }
    } catch (e) {
      setDisableButton(false);
      setCountdown(30);
    } finally {
      setSendingSms(false);
    }
  }

  async function handleSmsSubmit() {
    if (!/^1\d{10}$/.test(phone)) {
      showError(t('请输入正确的手机号'));
      return;
    }
    if (!/^\d{6}$/.test(verification_code)) {
      showError(t('请输入6位验证码'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setLoading(true);
    try {
      const res = await API.post(
        `/api/user/reset/sms?turnstile=${turnstileToken}`,
        {
          phone,
          verification_code,
        },
      );
      const { success, message, data } = res.data;
      if (success) {
        setNewPassword(data);
        showSuccess(t('密码重置成功，请妥善保存新密码'));
        if (data) {
          copy(data);
        }
      } else {
        showError(message || t('手机号或验证码错误'));
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className='classic-page-fill relative overflow-hidden bg-gray-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8'>
      <div
        className='blur-ball blur-ball-indigo'
        style={{ top: '-80px', right: '-80px', transform: 'none' }}
      />
      <div
        className='blur-ball blur-ball-teal'
        style={{ top: '50%', left: '-120px' }}
      />
      <div className='w-full max-w-sm mt-[60px]'>
        <div className='flex flex-col items-center'>
          <div className='w-full max-w-md'>
            <div className='flex items-center justify-center mb-6 gap-2'>
              <img src={logo} alt='Logo' className='h-10 rounded-full' />
              <Title heading={3} className='!text-gray-800'>
                {systemName}
              </Title>
            </div>

            <Card className='border-0 !rounded-2xl overflow-hidden'>
              <div className='flex justify-center pt-6 pb-2'>
                <Title heading={3} className='text-gray-800 dark:text-gray-200'>
                  {t('密码重置')}
                </Title>
              </div>
              <div className='px-2 py-8'>
                {newPassword ? (
                  <div className='space-y-4'>
                    <Text>{t('密码已重置，请使用以下新密码登录：')}</Text>
                    <Form.Input
                      field='new_password'
                      label={t('新密码')}
                      value={newPassword}
                      readonly
                    />
                    <Button
                      theme='solid'
                      className='w-full !rounded-full'
                      type='primary'
                      onClick={() => copy(newPassword)}
                    >
                      {t('复制密码')}
                    </Button>
                  </div>
                ) : (
                  <Form className='space-y-3'>
                    {smsResetEnabled ? (
                      <Tabs
                        type='line'
                        activeKey={activeTab}
                        onChange={setActiveTab}
                      >
                        <TabPane tab={t('邮箱')} itemKey='email' />
                        <TabPane tab={t('手机号')} itemKey='phone' />
                      </Tabs>
                    ) : null}

                    {activeTab === 'phone' && smsResetEnabled ? (
                      <>
                        <Form.Input
                          field='phone'
                          label={t('手机号')}
                          placeholder={t('请输入手机号')}
                          name='phone'
                          value={phone}
                          onChange={(v) => handleChange('phone', v)}
                          prefix={<IconPhone />}
                        />
                        <div className='flex gap-2 items-end'>
                          <div className='flex-1'>
                            <Form.Input
                              field='verification_code'
                              label={t('验证码')}
                              placeholder={t('请输入验证码')}
                              name='verification_code'
                              value={verification_code}
                              onChange={(v) =>
                                handleChange('verification_code', v)
                              }
                            />
                          </div>
                          <Button
                            onClick={handleSendSmsCode}
                            loading={sendingSms}
                            disabled={disableButton}
                          >
                            {disableButton
                              ? `${t('重试')} (${countdown})`
                              : t('发送验证码')}
                          </Button>
                        </div>
                        <div className='space-y-2 pt-2'>
                          <Button
                            theme='solid'
                            className='w-full !rounded-full'
                            type='primary'
                            onClick={handleSmsSubmit}
                            loading={loading}
                          >
                            {t('重置密码')}
                          </Button>
                        </div>
                      </>
                    ) : (
                      <>
                        <Form.Input
                          field='email'
                          label={t('邮箱')}
                          placeholder={t('请输入您的邮箱地址')}
                          name='email'
                          value={email}
                          onChange={(v) => handleChange('email', v)}
                          prefix={<IconMail />}
                        />
                        <div className='space-y-2 pt-2'>
                          <Button
                            theme='solid'
                            className='w-full !rounded-full'
                            type='primary'
                            htmlType='submit'
                            onClick={handleEmailSubmit}
                            loading={loading}
                            disabled={disableButton}
                          >
                            {disableButton
                              ? `${t('重试')} (${countdown})`
                              : t('提交')}
                          </Button>
                        </div>
                      </>
                    )}
                  </Form>
                )}

                <div className='mt-6 text-center text-sm'>
                  <Text>
                    {t('想起来了？')}{' '}
                    <Link
                      to='/login'
                      className='text-blue-600 hover:text-blue-800 font-medium'
                    >
                      {t('登录')}
                    </Link>
                  </Text>
                </div>
              </div>
            </Card>

            {turnstileEnabled && (
              <div className='flex justify-center mt-6'>
                <Turnstile
                  sitekey={turnstileSiteKey}
                  onVerify={(token) => {
                    setTurnstileToken(token);
                  }}
                />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default PasswordResetForm;
