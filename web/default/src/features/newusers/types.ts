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
export interface NewuserItem {
  id: number
  owner_user_id: number
  username: string
  display_name: string
  email?: string
  phone?: string
  status: number
  token_id: number
  quota_limit: number
  created_time: number
  last_login_time: number
  used_quota: number
  remain_quota: number
  unlimited: boolean
}

export interface NewuserListSummary {
  total_users: number
  active_users: number
  total_used: number
}

export interface NewuserListResponse {
  items: NewuserItem[]
  summary: NewuserListSummary
}

export interface NewuserSettings {
  enabled: boolean
  register_enabled: boolean
  register_code: string
  owner_user_id: number
}

export interface NewuserUsageLog {
  id: number
  created_at: number
  type: number
  content: string
  model_name?: string
  quota?: number
  prompt_tokens?: number
  completion_tokens?: number
}

export interface NewuserUsageDetail {
  user: NewuserItem
  quota_limit: number
  used_quota: number
  remain_quota: number
  unlimited: boolean
  owner_user_id?: number
  owner_username?: string
  owner_display_name?: string
  owner_quota?: number
  owner_used_quota?: number
  recent_logs: NewuserUsageLog[]
}

export interface NewuserFormData {
  username: string
  password: string
  display_name: string
  email?: string
  phone?: string
  quota_limit: number
}

export interface NewuserUpdateData {
  display_name?: string
  email?: string
  phone?: string
  status?: number
  quota_limit?: number
  password?: string
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export type NewusersDialogType = 'create' | 'update' | 'delete' | 'usage'
