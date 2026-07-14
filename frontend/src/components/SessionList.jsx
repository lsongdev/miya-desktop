import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import {
  CreateSession,
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
} from 'lucide-react'

export default function SessionList({ activeSessionId, onSelectSession, onNewSession: onNewSessionProp, onRefresh, refreshKey, connected }) {
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)

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

  const handleNewSession = async () => {
    setCreating(true)
    try {
      const session = await CreateSession('/tmp')
      await fetchSessions()
      ;(onNewSessionProp || onSelectSession)(session)
    } catch (err) {
      console.error('Create session error:', err)
    } finally {
      setCreating(false)
    }
  }

  const handleClose = async (sessionId, e) => {
    e.stopPropagation()
    try {
      await CloseSession(sessionId)
      await fetchSessions()
    } catch (err) {
      console.error('Close session error:', err)
    }
  }

  const handleDelete = async (sessionId, e) => {
    e.stopPropagation()
    try {
      await DeleteSession(sessionId)
      await fetchSessions()
    } catch (err) {
      console.error('Delete session error:', err)
    }
  }

  return (
    <div className="flex flex-col h-full border-r bg-card">
      <div className="shrink-0 p-3 border-b">
        <Button
          className="w-full"
          size="sm"
          onClick={handleNewSession}
          disabled={creating || !connected}
        >
          {creating ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <Plus className="size-3" />
          )}
          New Session
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {!connected ? (
          <div className="py-8 px-3 text-center text-xs text-muted-foreground">
            Connect an agent to view sessions
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
