import React, { createContext, useContext, useMemo, useState } from 'react'

export type AccountRef = { id: string; name?: string }

type SelectionState = {
  accounts: AccountRef[]
  selectedAccountId?: string
  selectedServiceCode?: string
  setSelectedAccountId: (id?: string) => void
  setSelectedServiceCode: (code?: string) => void
  upsertAccount: (a: AccountRef) => void
  removeAccount: (id: string) => void
}

const STORAGE_KEY = 'digit.demoUi.selection.v1'

function loadState(): { accounts: AccountRef[]; selectedAccountId?: string; selectedServiceCode?: string } {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return { accounts: [] }
  try {
    const parsed = JSON.parse(raw) as any
    return {
      accounts: Array.isArray(parsed.accounts) ? (parsed.accounts as AccountRef[]) : [],
      selectedAccountId: typeof parsed.selectedAccountId === 'string' ? parsed.selectedAccountId : undefined,
      selectedServiceCode: typeof parsed.selectedServiceCode === 'string' ? parsed.selectedServiceCode : undefined,
    }
  } catch {
    return { accounts: [] }
  }
}

function saveState(s: { accounts: AccountRef[]; selectedAccountId?: string; selectedServiceCode?: string }) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(s))
}

const Ctx = createContext<SelectionState | null>(null)

export function SelectionProvider(props: { children: React.ReactNode }) {
  const initial = useMemo(() => loadState(), [])
  const [accounts, setAccounts] = useState<AccountRef[]>(initial.accounts || [])
  const [selectedAccountId, setSelectedAccountId_] = useState<string | undefined>(initial.selectedAccountId)
  const [selectedServiceCode, setSelectedServiceCode_] = useState<string | undefined>(initial.selectedServiceCode)

  function persist(next: { accounts: AccountRef[]; selectedAccountId?: string; selectedServiceCode?: string }) {
    saveState(next)
  }

  function setSelectedAccountId(id?: string) {
    setSelectedAccountId_(id)
    const next = { accounts, selectedAccountId: id, selectedServiceCode }
    persist(next)
  }

  function setSelectedServiceCode(code?: string) {
    setSelectedServiceCode_(code)
    const next = { accounts, selectedAccountId, selectedServiceCode: code }
    persist(next)
  }

  function upsertAccount(a: AccountRef) {
    setAccounts((prev) => {
      const next = [...prev.filter((x) => x.id !== a.id), a].sort((x, y) => x.id.localeCompare(y.id))
      persist({ accounts: next, selectedAccountId, selectedServiceCode })
      return next
    })
  }

  function removeAccount(id: string) {
    setAccounts((prev) => {
      const next = prev.filter((x) => x.id !== id)
      const nextSelected = selectedAccountId === id ? undefined : selectedAccountId
      setSelectedAccountId_(nextSelected)
      persist({ accounts: next, selectedAccountId: nextSelected, selectedServiceCode })
      return next
    })
  }

  const value: SelectionState = useMemo(
    () => ({
      accounts,
      selectedAccountId,
      selectedServiceCode,
      setSelectedAccountId,
      setSelectedServiceCode,
      upsertAccount,
      removeAccount,
    }),
    [accounts, selectedAccountId, selectedServiceCode],
  )

  return <Ctx.Provider value={value}>{props.children}</Ctx.Provider>
}

export function useSelection(): SelectionState {
  const v = useContext(Ctx)
  if (!v) throw new Error('useSelection must be used within SelectionProvider')
  return v
}

