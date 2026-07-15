import { createContext, useContext, useState, useCallback, useEffect, useMemo } from 'react'
import { ConnectAgent, InitializeAgent, DisconnectAgent } from '../../wailsjs/go/main/App'
import { useMiyaConfig } from './MiyaConfigContext'

const AgentContext = createContext(null)

const SELECTED_KEY = 'miya-selected-agent'

const DEFAULT_AGENTS = [
  { id: 'miya', name: 'Miya Agents', type: 'stdio', command: 'miya', args: ['acp'] },
]

function commandFromAgent(agent) {
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
  const { config } = useMiyaConfig()
  const [selectedAgentId, setSelectedAgentIdState] = useState(loadSelectedId)
  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [agentInfo, setAgentInfo] = useState(null)
  const [error, setError] = useState(null)

  const agents = useMemo(() => {
    const configured = Array.isArray(config.agents) ? config.agents : []
    const source = configured.length > 0 ? configured : DEFAULT_AGENTS
    return source
      .filter((agent) => agent.enabled !== false)
      .map(normalizeAgent)
      .filter((agent) => agent.id && agent.command)
  }, [config.agents])

  const selectedAgent = agents.find((a) => a.id === selectedAgentId) || agents[0] || null

  const setSelectedAgentId = useCallback((id) => {
    setSelectedAgentIdState(id)
    saveSelectedId(id)
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
      if (target.id) setSelectedAgentId(target.id)
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
