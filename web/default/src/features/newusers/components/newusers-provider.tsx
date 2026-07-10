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
import React, { useCallback, useState } from 'react'

import useDialogState from '@/hooks/use-dialog'

import type { NewuserItem, NewusersDialogType } from '../types'

type NewusersContextType = {
  open: NewusersDialogType | null
  setOpen: (value: NewusersDialogType | null) => void
  currentRow: NewuserItem | null
  setCurrentRow: React.Dispatch<React.SetStateAction<NewuserItem | null>>
  refreshTrigger: number
  triggerRefresh: () => void
}

const NewusersContext = React.createContext<NewusersContextType | null>(null)

export function NewusersProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useDialogState<NewusersDialogType>(null)
  const [currentRow, setCurrentRow] = useState<NewuserItem | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const triggerRefresh = useCallback(() => {
    setRefreshTrigger((prev) => prev + 1)
  }, [])

  return (
    <NewusersContext
      value={{
        open,
        setOpen,
        currentRow,
        setCurrentRow,
        refreshTrigger,
        triggerRefresh,
      }}
    >
      {children}
    </NewusersContext>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useNewusers() {
  const context = React.useContext(NewusersContext)
  if (!context) {
    throw new Error('useNewusers must be used within NewusersProvider')
  }
  return context
}
