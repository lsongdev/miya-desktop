import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { LoadMiyaConfig, MiyaConfigPath, SaveMiyaConfig } from '../../wailsjs/go/main/App'

const MiyaConfigContext = createContext(null)

const emptyConfig = {
  agents: {},
  providers: {},
  mcpServers: {},
  channels: {},
}

function normalizeConfig(config) {
  return {
    ...emptyConfig,
    ...(config || {}),
    agents: config?.agents || {},
    providers: config?.providers || {},
    mcpServers: config?.mcpServers || {},
    channels: config?.channels || {},
  }
}

export function MiyaConfigProvider({ children }) {
  const [config, setConfig] = useState(emptyConfig)
  const [path, setPath] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [nextConfig, nextPath] = await Promise.all([LoadMiyaConfig(), MiyaConfigPath()])
      setConfig(normalizeConfig(nextConfig))
      setPath(nextPath)
    } catch (err) {
      setError(err.toString())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    reload()
  }, [reload])

  const saveConfig = useCallback(async (updater) => {
    setSaving(true)
    setError(null)
    try {
      const next = typeof updater === 'function' ? normalizeConfig(updater(config)) : normalizeConfig(updater)
      await SaveMiyaConfig(next)
      setConfig(next)
      return next
    } catch (err) {
      setError(err.toString())
      throw err
    } finally {
      setSaving(false)
    }
  }, [config])

  const value = useMemo(() => ({
    config,
    path,
    loading,
    saving,
    error,
    reload,
    saveConfig,
  }), [config, path, loading, saving, error, reload, saveConfig])

  return (
    <MiyaConfigContext.Provider value={value}>
      {children}
    </MiyaConfigContext.Provider>
  )
}

export function useMiyaConfig() {
  const ctx = useContext(MiyaConfigContext)
  if (!ctx) throw new Error('useMiyaConfig must be used within MiyaConfigProvider')
  return ctx
}
