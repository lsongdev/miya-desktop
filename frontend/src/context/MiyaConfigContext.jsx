import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { LoadMiyaConfig, MiyaConfigExists, MiyaConfigPath, SaveMiyaConfig } from '../../bindings/wails-app/app'

const MiyaConfigContext = createContext(null)

const emptyConfig = {
  agents: [],
  profiles: {},
  providers: {},
  mcpServers: {},
  channels: [],
}

function normalizeConfig(config) {
  return {
    ...emptyConfig,
    ...(config || {}),
    agents: Array.isArray(config?.agents) ? config.agents : [],
    profiles: config?.profiles || {},
    providers: config?.providers || {},
    mcpServers: config?.mcpServers || {},
    channels: Array.isArray(config?.channels) ? config.channels : [],
  }
}

export function MiyaConfigProvider({ children }) {
  const [config, setConfig] = useState(emptyConfig)
  const [path, setPath] = useState('')
  const [exists, setExists] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [nextConfig, nextPath, configExists] = await Promise.all([
        LoadMiyaConfig(),
        MiyaConfigPath(),
        MiyaConfigExists(),
      ])
      setConfig(normalizeConfig(nextConfig))
      setPath(nextPath)
      setExists(configExists)
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
      setExists(true)
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
    exists,
    loading,
    saving,
    error,
    reload,
    saveConfig,
  }), [config, path, exists, loading, saving, error, reload, saveConfig])

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
