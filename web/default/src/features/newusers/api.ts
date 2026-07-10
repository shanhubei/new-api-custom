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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  NewuserFormData,
  NewuserListResponse,
  NewuserSettings,
  NewuserUpdateData,
  NewuserUsageDetail,
  NewuserItem,
} from './types'

export async function getNewusers(): Promise<ApiResponse<NewuserListResponse>> {
  const res = await api.get('/api/newuser/admin/users')
  return res.data
}

export async function createNewuser(
  data: NewuserFormData
): Promise<ApiResponse<{ user: NewuserItem; token_id: number }>> {
  const res = await api.post('/api/newuser/admin/users', data)
  return res.data
}

export async function updateNewuser(
  id: number,
  data: NewuserUpdateData
): Promise<ApiResponse<NewuserItem>> {
  const res = await api.put(`/api/newuser/admin/users/${id}`, data)
  return res.data
}

export async function deleteNewuser(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/newuser/admin/users/${id}`)
  return res.data
}

export async function getNewuserSettings(): Promise<ApiResponse<NewuserSettings>> {
  const res = await api.get('/api/newuser/admin/settings')
  return res.data
}

export async function updateNewuserSettings(
  data: Pick<NewuserSettings, 'enabled' | 'register_enabled' | 'register_code'>
): Promise<ApiResponse> {
  const res = await api.put('/api/newuser/admin/settings', data)
  return res.data
}

export async function getNewuserUsage(
  id: number
): Promise<ApiResponse<NewuserUsageDetail>> {
  const res = await api.get(`/api/newuser/admin/users/${id}/usage`)
  return res.data
}
