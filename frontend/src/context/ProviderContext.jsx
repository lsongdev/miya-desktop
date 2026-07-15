import { createContext, useContext, useState, useCallback } from 'react'

const ProviderContext = createContext(null)

const PROVIDERS_KEY = 'miya-providers'

const DEFAULT_PROVIDERS = [
  { id: 'openai', name: 'OpenAI', apiKey: '', baseUrl: 'https://api.openai.com/v1' },
  { id: 'anthropic', name: 'Anthropic', apiKey: '', baseUrl: 'https://api.anthropic.com' },
]

function loadProviders() {
  try {
    const raw = localStorage.getItem(PROVIDERS_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return DEFAULT_PROVIDERS
}

function saveProviders(providers) {
  try { localStorage.setItem(PROVIDERS_KEY, JSON.stringify(providers)) } catch {}
}

export function ProviderProvider({ children }) {
  const [providers, setProviders] = useState(loadProviders)

  const addProvider = useCallback((provider) => {
    setProviders((prev) => {
      const next = [...prev, provider]
      saveProviders(next)
      return next
    })
  }, [])

  const updateProvider = useCallback((id, updates) => {
    setProviders((prev) => {
      const next = prev.map((p) => (p.id === id ? { ...p, ...updates } : p))
      saveProviders(next)
      return next
    })
  }, [])

  const removeProvider = useCallback((id) => {
    setProviders((prev) => {
      const next = prev.filter((p) => p.id !== id)
      saveProviders(next)
      return next
    })
  }, [])

  const getProvider = useCallback((id) => {
    return providers.find((p) => p.id === id) || null
  }, [providers])

  return (
    <ProviderContext.Provider value={{
      providers, addProvider, updateProvider, removeProvider, getProvider,
    }}>
      {children}
    </ProviderContext.Provider>
  )
}

export function useProvider() {
  const ctx = useContext(ProviderContext)
  if (!ctx) throw new Error('useProvider must be used within ProviderProvider')
  return ctx
}
