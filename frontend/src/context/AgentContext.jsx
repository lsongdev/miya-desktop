import { createContext, useContext, useState, useCallback, useEffect, useMemo } from 'react'
import { ConnectAgent, InitializeAgent, DisconnectAgent } from '../../wailsjs/go/main/App'
import { useMiyaConfig } from './MiyaConfigContext'

const AgentContext = createContext(null)

const AGENTS_KEY = 'miya-agents'
const SELECTED_KEY = 'miya-selected-agent'

const DEFAULT_AGENTS = [
  { id: 'opencode', name: 'OpenCode', command: 'opencode acp' },
]

function commandFromACP(acp) {
  const command = acp?.command?.trim()
  if (!command) return 'miya acp'
  return [command, ...(acp.args || [])].join(' ')
}

function loadAgents() {
  try {
    const raw = localStorage.getItem(AGENTS_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return DEFAULT_AGENTS
}

function saveAgents(agents) {
  try { localStorage.setItem(AGENTS_KEY, JSON.stringify(agents)) } catch {}
}

function loadSelectedId() {
  try { return localStorage.getItem(SELECTED_KEY) || 'opencode' } catch { return 'opencode' }
}

function saveSelectedId(id) {
  try { localStorage.setItem(SELECTED_KEY, id) } catch {}
}

export function AgentProvider({ children }) {
  const { config } = useMiyaConfig()
  const [localAgents, setLocalAgents] = useState(loadAgents)
  const [selectedAgentId, setSelectedAgentIdState] = useState(loadSelectedId)
  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [agentInfo, setAgentInfo] = useState(null)
  const [error, setError] = useState(null)

  const agents = useMemo(() => {
    const miyaAgent = { id: 'miya-agents', name: 'Miya Agents', command: commandFromACP(config.acp) }
    return [miyaAgent, ...localAgents.filter((agent) => agent.id !== miyaAgent.id)]
  }, [config.acp, localAgents])

  const selectedAgent = agents.find((a) => a.id === selectedAgentId) || agents[0] || null

  const setSelectedAgentId = useCallback((id) => {
    setSelectedAgentIdState(id)
    saveSelectedId(id)
  }, [])

  const addAgent = useCallback((agent) => {
    setLocalAgents((prev) => {
      const next = [...prev, agent]
      saveAgents(next)
      return next
    })
  }, [])

  const updateAgent = useCallback((id, updates) => {
    setLocalAgents((prev) => {
      const next = prev.map((a) => (a.id === id ? { ...a, ...updates } : a))
      saveAgents(next)
      return next
    })
  }, [])

  const removeAgent = useCallback((id) => {
    setLocalAgents((prev) => {
      const next = prev.filter((a) => a.id !== id)
      saveAgents(next)
      return next
    })
  }, [])

  const connect = useCallback(async (agent) => {
    const target = agent || selectedAgent
    if (!target?.command?.trim()) return

    setConnecting(true)
    setError(null)
    try {
      await DisconnectAgent().catch(() => {})
      await ConnectAgent(target.command)
      const info = await InitializeAgent('miya-desktop', '0.1.0')
      setAgentInfo(info)
      setConnected(true)
    } catch (err) {
      setError(err.toString())
      setConnected(false)
      setAgentInfo(null)
    } finally {
      setConnecting(false)
    }
  }, [selectedAgent])

  const disconnect = useCallback(async () => {
    try { await DisconnectAgent() } catch {}
    setConnected(false)
    setAgentInfo(null)
  }, [])

  const reconnect = useCallback(async (agent) => {
    await disconnect()
    await connect(agent)
  }, [disconnect, connect])

  useEffect(() => {
    if (selectedAgent?.command && !connected && !connecting) {
      connect()
    }
  }, [])

  return (
    <AgentContext.Provider value={{
      agents, selectedAgent, selectedAgentId, setSelectedAgentId,
      addAgent, updateAgent, removeAgent,
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
