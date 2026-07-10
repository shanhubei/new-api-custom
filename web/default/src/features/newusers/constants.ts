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
export const NEWUSER_STATUS = {
  ENABLED: 1,
  DISABLED: 2,
} as const

export const SUCCESS_MESSAGES = {
  CREATED: 'Third-party user created successfully',
  UPDATED: 'Third-party user updated successfully',
  DELETED: 'Third-party user disabled successfully',
  SETTINGS_SAVED: 'Third-party user settings saved',
} as const

export const ERROR_MESSAGES = {
  UNEXPECTED: 'An unexpected error occurred',
  MODULE_DISABLED: 'Third-party user module is disabled on this server',
  ORG_DISABLED: 'Third-party users are not enabled for your organization',
  DELETE_FAILED: 'Failed to disable third-party user',
} as const
