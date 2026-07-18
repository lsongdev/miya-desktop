import { createContext, useContext, useState, useCallback, useEffect, useMemo } from 'react'
import { ConnectAgent, ConnectConfiguredAgent, InitializeAgent, DisconnectAgent } from '../../bindings/wails-app/app'
import { useMiyaConfig } from './MiyaConfigContext'

const AgentContext = createContext(null)

const SELECTED_KEY = 'miya-selected-agent'

function commandFromAgent(agent) {
  if (agent?.type === 'builtin') return 'miya-agent acp'
  if (!agent?.command?.trim()) return ''
  return [agent.command.trim(), ...(agent.args || [])].join(' ')
}

function normalizeAgent(agent) {
  const id = agent.id || agent.name || agent.command
  return {
    ...agent,
    id,
    name: agent.name || id,
    command: commandFromAgent(agent),
  }
}

function loadSelectedId() {
  try { return localStorage.getItem(SELECTED_KEY) || '' } catch { return '' }
}

function saveSelectedId(id) {
  try { localStorage.setItem(SELECTED_KEY, id) } catch {}
}

export function AgentProvider({ children }) {
  const { config, loading } = useMiyaConfig()
  const [selectedAgentId, setSelectedAgentIdState] = useState(loadSelectedId)
  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [agentInfo, setAgentInfo] = useState(null)
  const [error, setError] = useState(null)

  const agents = useMemo(() => {
    const profiles = Object.entries(config.profiles || {}).sort(([left], [right]) => {
      if (left === 'default') return -1
      if (right === 'default') return 1
      return left.localeCompare(right)
    })
    const builtinAgents = profiles.map(([id, profile]) => normalizeAgent({
      id,
      name: id,
      type: 'builtin',
      profile: id,
      model: profile?.model || '',
    }))
    const profileIDs = new Set(profiles.map(([id]) => id))
    const externalAgents = (Array.isArray(config.agents) ? config.agents : [])
      .filter((agent) => agent.enabled !== false && agent.type !== 'builtin')
      .map(normalizeAgent)
      .filter((agent) => agent.id && agent.command && !profileIDs.has(agent.id))
    return [...builtinAgents, ...externalAgents]
  }, [config.agents, config.profiles])

  const selectedAgent = agents.find((a) => a.id === selectedAgentId) || agents[0] || null

  const setSelectedAgentId = useCallback((id) => {
    setSelectedAgentIdState(id)
    saveSelectedId(id)
  }, [])

  const connectOne = useCallback(async (target) => {
    if (target?.id) {
      await ConnectConfiguredAgent(target.id)
    } else {
      await ConnectAgent(target.command)
    }
    const info = await InitializeAgent('miya-desktop', import.meta.env.VITE_APP_VERSION || 'dev')
    if (target.id) setSelectedAgentId(target.id)
    setAgentInfo(info)
    setConnected(true)
  }, [setSelectedAgentId])

  const connect = useCallback(async (agent, options = {}) => {
    const target = agent || selectedAgent
    if (!target || (target.type !== 'builtin' && !target.command?.trim())) return

    setConnecting(true)
    setError(null)
    try {
      await DisconnectAgent().catch(() => {})
      try {
        await connectOne(target)
      } catch (err) {
        if (!options.fallback) throw err
        const fallbackAgents = agents.filter((candidate) => candidate.id !== target.id)
        let lastErr = err
        for (const fallbackAgent of fallbackAgents) {
          try {
            await DisconnectAgent().catch(() => {})
            await connectOne(fallbackAgent)
            return
          } catch (fallbackErr) {
            lastErr = fallbackErr
          }
        }
        throw lastErr
      }
    } catch (err) {
      setError(err.toString())
      setConnected(false)
      setAgentInfo(null)
      if (options.throwOnError) throw err
    } finally {
      setConnecting(false)
    }
  }, [agents, connectOne, selectedAgent])

  const disconnect = useCallback(async () => {
    try { await DisconnectAgent() } catch {}
    setConnected(false)
    setAgentInfo(null)
  }, [])

  const reconnect = useCallback(async (agent) => {
    await disconnect()
    await connect(agent, { throwOnError: true })
  }, [disconnect, connect])

  useEffect(() => {
    if (loading || connecting) return
    if (!selectedAgent) {
      if (connected) disconnect()
      return
    }
    if (!connected || selectedAgent.id !== selectedAgentId) {
      connect(selectedAgent, { fallback: true })
    }
  }, [loading, selectedAgent?.id, selectedAgent?.command, selectedAgentId, connected, connecting, connect, disconnect])

  return (
    <AgentContext.Provider value={{
      agents, selectedAgent, selectedAgentId, setSelectedAgentId,
      connected, connecting, agentInfo, error,
      connect, disconnect, reconnect,
    }}>
      {children}
    </AgentContext.Provider>
  )
}

export function useAgent() {
  const ctx = useContext(AgentContext)
  if (!ctx) throw new Error('useAgent must be used within AgentProvider')
  return ctx
}
