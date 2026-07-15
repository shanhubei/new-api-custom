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

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Button, Input, Modal } from '@douyinfe/semi-ui';
import { IconPhone, IconKey } from '@douyinfe/semi-icons';

const PhoneBindModal = ({
  t,
  showPhoneBindModal,
  setShowPhoneBindModal,
  inputs,
  handleInputChange,
  sendSmsBindCode,
  bindPhone,
  disableButton,
  loading,
  countdown,
}) => {
  return (
    <Modal
      title={
        <div className='flex items-center'>
          <IconPhone className='mr-2 text-blue-500' />
          {t('绑定手机号')}
        </div>
      }
      visible={showPhoneBindModal}
      onCancel={() => setShowPhoneBindModal(false)}
      onOk={bindPhone}
      size={'small'}
      centered={true}
      maskClosable={false}
      className='modern-modal'
      okButtonProps={{ loading }}
    >
      <div className='space-y-4 py-4'>
        <div className='flex gap-3'>
          <Input
            placeholder={t('输入手机号')}
            onChange={(value) => handleInputChange('phone', value)}
            name='phone'
            type='tel'
            size='large'
            className='!rounded-lg flex-1'
            prefix={<IconPhone />}
            maxLength={11}
          />
          <Button
            onClick={sendSmsBindCode}
            disabled={disableButton || loading}
            className='!rounded-lg'
            type='primary'
            theme='outline'
            size='large'
          >
            {disableButton
              ? `${t('重新发送')} (${countdown})`
              : t('获取验证码')}
          </Button>
        </div>

        <Input
          placeholder={t('验证码')}
          name='phone_verification_code'
          value={inputs.phone_verification_code}
          onChange={(value) =>
            handleInputChange('phone_verification_code', value)
          }
          size='large'
          className='!rounded-lg'
          prefix={<IconKey />}
          maxLength={6}
        />
      </div>
    </Modal>
  );
};

export default PhoneBindModal;
