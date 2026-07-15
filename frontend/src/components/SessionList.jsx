import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import {
  CreateSession,
  DefaultCwd,
  ListSessions,
  CloseSession,
  DeleteSession,
} from '../../wailsjs/go/main/App'
import {
  Plus,
  Loader2,
  MessageSquare,
  Trash2,
  X,
  ChevronDown,
} from 'lucide-react'

export default function SessionList({ activeSessionId, onSelectSession, onNewSession: onNewSessionProp, onRefresh, refreshKey, connected, agents = [], currentAgentId, onBeforeCreateSession, onSessionClosed, onSessionDeleted }) {
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [actionError, setActionError] = useState(null)
  const [menuOpen, setMenuOpen] = useState(false)

  const fetchSessions = useCallback(async () => {
    try {
      setLoading(true)
      const list = await ListSessions()
      setSessions(list || [])
    } catch (err) {
      console.error('Failed to list sessions:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (connected) {
      fetchSessions()
    } else {
      setSessions([])
    }
  }, [connected, fetchSessions, refreshKey])

  useEffect(() => {
    if (onRefresh) onRefresh.current = fetchSessions
  }, [fetchSessions, onRefresh])

  const handleNewSession = async (agent) => {
    setCreating(true)
    setActionError(null)
    setMenuOpen(false)
    try {
      await onBeforeCreateSession?.(agent)
      const cwd = await DefaultCwd()
      const session = await CreateSession(cwd)
      await fetchSessions()
      ;(onNewSessionProp || onSelectSession)(session)
    } catch (err) {
      console.error('Create session error:', err)
      setActionError(`Create failed: ${err?.toString() || err}`)
    } finally {
      setCreating(false)
    }
  }

  const handleClose = async (sessionId, e) => {
    e.stopPropagation()
    setActionError(null)
    try {
      await CloseSession(sessionId)
      await fetchSessions()
      onSessionClosed?.(sessionId)
    } catch (err) {
      console.error('Close session error:', err)
      setActionError(`Close failed: ${err?.toString() || err}`)
    }
  }

  const handleDelete = async (sessionId, e) => {
    e.stopPropagation()
    setActionError(null)
    // Optimistic removal so the row disappears immediately; if the agent
    // rejects the delete we restore the list and surface the error.
    const snapshot = sessions
    setSessions((prev) => prev.filter((s) => s.id !== sessionId))
    try {
      await DeleteSession(sessionId)
      onSessionDeleted?.(sessionId)
      // Refresh from server so title/updatedAt on remaining rows stays fresh.
      fetchSessions()
    } catch (err) {
      console.error('Delete session error:', err)
      setSessions(snapshot)
      setActionError(`Delete failed: ${err?.toString() || err}`)
    }
  }

  return (
    <div className="flex flex-col h-full border-r bg-card">
      <div className="shrink-0 p-3 border-b">
        <div className="relative flex">
          <Button
            className="flex-1 rounded-r-none"
            size="sm"
            onClick={() => handleNewSession(agents.find((agent) => agent.id === currentAgentId) || agents[0])}
            disabled={creating || agents.length === 0}
          >
            {creating ? <Loader2 className="size-3 animate-spin" /> : <Plus className="size-3" />}
            New Session
          </Button>
          <Button
            size="icon-sm"
            variant="outline"
            className="rounded-l-none border-l-0"
            onClick={() => setMenuOpen((open) => !open)}
            disabled={creating || agents.length <= 1}
            title="Choose agent"
          >
            <ChevronDown className="size-3.5" />
          </Button>
          {menuOpen && (
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
        <div className="shrink-0 px-3 py-2 text-[10px] text-destructive bg-destructive/10 border-b flex items-start gap-1">
          <span className="flex-1 break-words">{actionError}</span>
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
        {!connected ? (
          <div className="py-8 px-3 text-center text-xs text-muted-foreground">
            Create a session to connect an agent
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
            {sessions.map((session) => (
              <div
                key={session.id}
                onClick={() => onSelectSession(session)}
                className={`group flex items-center gap-2 px-3 py-2.5 cursor-pointer transition-colors hover:bg-muted/50 text-sm ${
                  activeSessionId === session.id ? 'bg-muted' : ''
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
                    onClick={(e) => handleClose(session.id, e)}
                    title="Close"
                  >
                    <X className="size-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    onClick={(e) => handleDelete(session.id, e)}
                    title="Delete"
                  >
                    <Trash2 className="size-3" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
