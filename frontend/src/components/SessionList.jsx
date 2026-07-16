import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import {
  CreateSession,
  DefaultCwd,
  ListAgentSessions,
  CloseSession,
  DeleteSession,
} from '../../bindings/wails-app/app'
import {
  Plus,
  Loader2,
  MessageSquare,
  Trash2,
  X,
  ChevronDown,
} from 'lucide-react'
import { useNavigate } from '../hooks/useNavigate'

function sessionKey(session) {
  return session?.key || session?.id || ''
}

function mergeSessions(primary = [], preserved = []) {
  const merged = new Map()
  for (const session of primary) {
    const key = sessionKey(session)
    if (key) merged.set(key, session)
  }
  for (const session of preserved) {
    const key = sessionKey(session)
    if (key && !merged.has(key)) merged.set(key, session)
  }
  return Array.from(merged.values())
}

function groupSessions(sessions, agents) {
  const groups = []
  const byAgent = new Map()
  const agentNames = new Map(agents.map((agent) => [agent.id, agent.name || agent.id]))
  for (const session of sessions) {
    const agentId = session.agentId || 'unknown'
    if (!byAgent.has(agentId)) {
      const group = {
        id: agentId,
        name: agentNames.get(agentId) || session.agentName || agentId,
        sessions: [],
      }
      byAgent.set(agentId, group)
      groups.push(group)
    }
    byAgent.get(agentId).sessions.push(session)
  }
  return groups
}

export default function SessionList({ activeSessionId, onSelectSession, onNewSession: onNewSessionProp, onRefresh, refreshKey, agents = [], currentAgentId, onBeforeCreateSession, onBeforeSessionAction, onSessionClosed, onSessionDeleted }) {
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [actionError, setActionError] = useState(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const navigate = useNavigate()

  const fetchSessions = useCallback(async (preserve = []) => {
    try {
      setLoading(true)
      const list = await ListAgentSessions()
      setSessions(mergeSessions(list || [], preserve))
    } catch (err) {
      console.error('Failed to list sessions:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (agents.length > 0) {
      fetchSessions()
    } else {
      setSessions([])
    }
  }, [agents, fetchSessions, refreshKey])

  useEffect(() => {
    if (onRefresh) onRefresh.current = fetchSessions
  }, [fetchSessions, onRefresh])

  const handleNewSession = async (agent) => {
    setCreating(true)
    setActionError(null)
    setMenuOpen(false)
    try {
      const targetAgent = await onBeforeCreateSession?.(agent) || agent
      const cwd = await DefaultCwd()
      const session = await CreateSession(cwd)
      if (!session?.id) throw new Error('Agent returned an empty session')
      const enrichedSession = targetAgent ? {
        ...session,
        agentId: targetAgent.id,
        agentName: targetAgent.name || targetAgent.id,
        agentCommand: targetAgent.command,
        key: `${targetAgent.id}:${session.id}`,
      } : session
      setSessions((prev) => mergeSessions(prev, [enrichedSession]))
      ;(onNewSessionProp || onSelectSession)(enrichedSession)
      await fetchSessions([enrichedSession])
    } catch (err) {
      console.error('Create session error:', err)
      setActionError(`Create failed: ${err?.toString() || err}`)
    } finally {
      setCreating(false)
    }
  }

  const handleSelect = async (session) => {
    setActionError(null)
    try {
      await onSelectSession(session)
    } catch (err) {
      console.error('Select session error:', err)
      setActionError(`Select failed: ${err?.toString() || err}`)
    }
  }

  const handleClose = async (session, e) => {
    e.stopPropagation()
    setActionError(null)
    try {
      await onBeforeSessionAction?.(session)
      await CloseSession(session.id)
      await fetchSessions()
      onSessionClosed?.(sessionKey(session))
    } catch (err) {
      console.error('Close session error:', err)
      setActionError(`Close failed: ${err?.toString() || err}`)
    }
  }

  const handleDelete = async (session, e) => {
    e.stopPropagation()
    setActionError(null)
    const key = sessionKey(session)
    // Optimistic removal so the row disappears immediately; if the agent
    // rejects the delete we restore the list and surface the error.
    const snapshot = sessions
    setSessions((prev) => prev.filter((s) => sessionKey(s) !== key))
    try {
      await onBeforeSessionAction?.(session)
      await DeleteSession(session.id)
      onSessionDeleted?.(key)
      // Refresh from server so title/updatedAt on remaining rows stays fresh.
      fetchSessions()
    } catch (err) {
      console.error('Delete session error:', err)
      setSessions(snapshot)
      setActionError(`Delete failed: ${err?.toString() || err}`)
    }
  }

  const groups = groupSessions(sessions, agents)
  const showAgentMenu = agents.length > 1
  const missingProfiles = actionError?.includes('no profiles configured')
  const actionErrorMessage = missingProfiles
    ? 'No agent profiles configured.'
    : actionError

  return (
    <div className="flex flex-col h-full border-r bg-card">
      <div className="shrink-0 p-3 border-b">
        <div className="relative flex">
          <Button
            className={`flex-1 ${showAgentMenu ? 'rounded-r-none border-r border-primary-foreground/20' : ''}`}
            size="sm"
            onClick={() => handleNewSession(agents.find((agent) => agent.id === currentAgentId) || agents[0])}
            disabled={creating || agents.length === 0}
          >
            {creating ? <Loader2 className="size-3 animate-spin" /> : <Plus className="size-3" />}
            New Session
          </Button>
          {showAgentMenu && (
            <Button
              size="icon-sm"
              className="rounded-l-none border-l-0 px-2"
              onClick={() => setMenuOpen((open) => !open)}
              disabled={creating}
              title="Choose agent"
            >
              <ChevronDown className="size-3.5" />
            </Button>
          )}
          {showAgentMenu && menuOpen && (
            <div className="absolute left-0 right-0 top-8 z-20 rounded-md border bg-popover p-1 shadow-md">
              {agents.map((agent) => (
                <button
                  key={agent.id}
                  onClick={() => handleNewSession(agent)}
                  className={`flex w-full min-w-0 flex-col rounded px-2 py-1.5 text-left text-xs hover:bg-muted ${
                    agent.id === currentAgentId ? 'bg-muted' : ''
                  }`}
                >
                  <span className="truncate font-medium">{agent.name || agent.id}</span>
                  <span className="truncate font-mono text-[10px] text-muted-foreground">{agent.command}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {actionError && (
        <div className="shrink-0 px-3 py-2 text-[10px] text-destructive bg-destructive/10 border-b flex items-start gap-1.5">
          <div className="flex-1 min-w-0">
            <span className="break-words">{actionErrorMessage}</span>
            {missingProfiles && (
              <button
                className="ml-1 font-medium underline underline-offset-2 hover:text-destructive/80"
                onClick={() => navigate('settings', { settingsTab: 'profiles' })}
              >
                Go profiles settings →
              </button>
            )}
          </div>
          <button
            onClick={() => setActionError(null)}
            className="shrink-0 opacity-70 hover:opacity-100"
            aria-label="Dismiss error"
          >
            <X className="size-3" />
          </button>
        </div>
      )}
      <div className="flex-1 overflow-y-auto">
        {agents.length === 0 ? (
          <div className="py-8 px-3 text-center text-xs text-muted-foreground">
            Enable an agent in Settings
          </div>
        ) : loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="size-4 animate-spin text-muted-foreground" />
          </div>
        ) : sessions.length === 0 ? (
          <div className="py-8 px-3 text-center text-xs text-muted-foreground">
            No sessions yet
          </div>
        ) : (
          <div className="divide-y">
            {groups.map((group) => (
              <div key={group.id}>
                <div className="sticky top-0 z-10 border-b bg-card/95 px-3 py-1.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                  {group.name}
                </div>
                {group.sessions.map((session) => {
                  const key = sessionKey(session)
                  return (
                    <div
                      key={key}
                      onClick={() => handleSelect(session)}
                      className={`group flex items-center gap-2 px-3 py-2.5 cursor-pointer transition-colors hover:bg-muted/50 text-sm ${
                        activeSessionId === key ? 'bg-muted' : ''
                      }`}
                    >
                      <MessageSquare className="size-3.5 shrink-0 text-muted-foreground" />
                      <div className="flex-1 min-w-0">
                        <p className="truncate text-xs font-medium">
                          {session.title || `Session ${session.id.slice(0, 8)}...`}
                        </p>
                        <p className="truncate text-[10px] text-muted-foreground">
                          {session.id.slice(0, 16)}...
                        </p>
                      </div>
                      <div className="flex items-center shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={(e) => handleClose(session, e)}
                          title="Close"
                        >
                          <X className="size-3" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={(e) => handleDelete(session, e)}
                          title="Delete"
                        >
                          <Trash2 className="size-3" />
                        </Button>
                      </div>
                    </div>
                  )
                })}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
