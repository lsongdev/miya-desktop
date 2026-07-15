import { useState, useEffect, useRef, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import SessionList from '@/components/SessionList'
import MarkdownContent from '@/components/MarkdownContent'
import { useAgent } from '@/context/AgentContext'
import { CancelSession, DefaultCwd, SendPrompt, LoadSession } from '../../bindings/wails-app/app'
import { Events } from '@wailsio/runtime'
import {
  Send,
  Square,
  Loader2,
  User,
  Bot,
  Brain,
  Wrench,
  CheckCircle2,
  XCircle,
  Clock,
  AlertTriangle,
} from 'lucide-react'

const StopReasonLabels = {
  end_turn: 'Completed',
  max_tokens: 'Max tokens reached',
  max_turn_requests: 'Max turn requests reached',
  refusal: 'Request refused',
  cancelled: 'Cancelled',
}

function ToolCallDisplay({ tool }) {
  if (!tool) return null
  return (
    <div className="flex items-start gap-2 text-xs text-muted-foreground bg-muted/30 rounded-md p-2 my-1 max-w-full">
      <Wrench className="size-3 mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="font-medium text-foreground/80 truncate">{tool.title || tool.kind}</p>
        {tool.content?.map((c, i) => (
          <p key={i} className="truncate">
            {c.content?.text || c.content?.content || c.newText || c.path || `${c.type} content`}
          </p>
        ))}
      </div>
      {tool.status === 'completed' && <CheckCircle2 className="size-3 shrink-0 text-green-500" />}
      {tool.status === 'failed' && <XCircle className="size-3 shrink-0 text-destructive" />}
      {tool.status === 'in_progress' && <Loader2 className="size-3 shrink-0 animate-spin" />}
      {tool.status === 'pending' && <Clock className="size-3 shrink-0" />}
    </div>
  )
}

function ThoughtDisplay({ text }) {
  if (!text) return null
  return (
    <div className="flex items-start gap-2 text-xs text-muted-foreground bg-muted/20 rounded-md p-2 my-1 italic border-l-2 border-muted-foreground/20 max-w-full">
      <Brain className="size-3 mt-0.5 shrink-0" />
      <span className="min-w-0 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">{text}</span>
    </div>
  )
}

function PlanDisplay({ plan }) {
  if (!plan?.entries?.length) return null
  return (
    <div className="text-xs text-muted-foreground bg-muted/20 rounded-md p-2 my-1 space-y-1 max-w-full">
      {plan.entries.map((entry, i) => (
        <div key={i} className="flex items-start gap-2 min-w-0">
          <span className={`size-1.5 mt-1.5 shrink-0 rounded-full ${
            entry.status === 'completed' ? 'bg-green-500' :
            entry.status === 'in_progress' ? 'bg-blue-500 animate-pulse' :
            'bg-muted-foreground/30'
          }`} />
          <span className={`min-w-0 break-words [overflow-wrap:anywhere] ${
            entry.status === 'completed' ? 'line-through opacity-60' : ''
          }`}>
            {entry.content}
          </span>
        </div>
      ))}
    </div>
  )
}

function MessageBlock({ block, role, streaming }) {
  if (block.type === 'thought') return <ThoughtDisplay text={block.content} />
  if (block.type === 'tool_call') return <ToolCallDisplay tool={block.tool} />
  if (block.type === 'plan') return <PlanDisplay plan={block.plan} />
  if (block.type === 'image') {
    const src = block.data ? `data:${block.mime};base64,${block.data}` : null
    return src ? <img src={src} alt="" className="max-w-[280px] rounded-md border" /> : null
  }
  if (block.type === 'audio') {
    const src = block.data ? `data:${block.mime};base64,${block.data}` : null
    return src ? <audio src={src} controls className="max-w-full" /> : null
  }

  return (
    <div
      className={`rounded-lg px-3 py-2 text-sm break-words [overflow-wrap:anywhere] max-w-[85%] ${
        role === 'user'
          ? 'bg-primary text-primary-foreground whitespace-pre-wrap'
          : 'bg-muted/50'
      }`}
    >
      {role === 'user' || block.type === 'text' ? (
        block.content
      ) : (
        <MarkdownContent content={block.content} />
      )}
      {streaming && <span className="inline-block w-1.5 h-4 bg-current ml-0.5 animate-pulse align-middle" />}
    </div>
  )
}

function usageLabel(usage) {
  if (!usage?.size) return null
  const percent = Math.round((Number(usage.used || 0) / Number(usage.size)) * 100)
  return `${usage.used || 0}/${usage.size} (${percent}%)`
}

function sessionKey(session) {
  return session?.key || session?.id || ''
}

function ChatWindow({ sessionId, session, shouldLoad, onLoadComplete }) {
  const [conversation, setConversation] = useState(null)
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [stopReason, setStopReason] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const messagesEndRef = useRef(null)
  const loadedRef = useRef(false)
  const inputRef = useRef(null)

  useEffect(() => {
    if (shouldLoad && !loadedRef.current) {
      loadedRef.current = true
      setLoading(true)
      setConversation(null)
      Promise.resolve(session.cwd || DefaultCwd())
        .then((cwd) => LoadSession(sessionId, cwd))
        .catch((err) => {
          setError(err.toString())
        })
        .finally(() => {
          setLoading(false)
          onLoadComplete?.()
        })
    }
  }, [shouldLoad, sessionId, session?.cwd, onLoadComplete])

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [conversation?.messages, scrollToBottom])

  useEffect(() => {
    const cleanup = Events.On('conversation:update', (event) => {
      try {
        const snapshot = event?.data ?? event
        const next = snapshot?.conversation
        if (next?.id !== sessionId && next?.acpSessionId !== sessionId) return
        setConversation(next)
        if (snapshot.stopReason) setStopReason(snapshot.stopReason)
      } catch (err) {
        console.error('Failed to handle conversation update', err)
      }
    })

    return () => {
      cleanup()
    }
  }, [sessionId])

  const handleSend = async () => {
    const text = input.trim()
    if (!text || streaming) return

    setInput('')
    if (inputRef.current) inputRef.current.style.height = 'auto'
    setStreaming(true)
    setStopReason(null)
    setError(null)

    try {
      await SendPrompt(sessionId, text)
    } catch (err) {
      setError(err.toString())
    } finally {
      setStreaming(false)
    }
  }

  const handleCancel = async () => {
    if (!streaming) return
    try {
      await CancelSession(sessionId)
    } catch (err) {
      setError(err.toString())
    }
  }

  const handleInputChange = (e) => {
    setInput(e.target.value)
    e.target.style.height = 'auto'
    e.target.style.height = `${Math.min(e.target.scrollHeight, 144)}px`
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault()
      handleSend()
    }
  }

  const messages = conversation?.messages || []
  const usage = usageLabel(conversation?.usage)
  const mode = conversation?.mode?.currentModeId
  const cwd = conversation?.cwd || session?.cwd

  return (
    <div className="flex flex-col h-full min-h-0 min-w-0">
      <div className="flex-1 overflow-y-auto overflow-x-hidden space-y-4 mb-4 min-h-0 min-w-0 pr-1">
        {loading && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Loader2 className="size-6 mb-2 animate-spin" />
            <p className="text-sm">Loading session...</p>
          </div>
        )}
        {messages.length === 0 && !loading && !streaming && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Bot className="size-8 mb-2" />
            <p className="text-sm">Start a conversation with the agent</p>
          </div>
        )}

        {messages.map((msg) => (
          <div key={msg.id} className={`flex gap-3 min-w-0 ${msg.role === 'user' ? 'flex-row-reverse' : ''}`}>
            <div
              className={`flex size-8 shrink-0 items-center justify-center rounded-full ${
                msg.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
              }`}
            >
              {msg.role === 'user' ? <User className="size-4" /> : <Bot className="size-4" />}
            </div>

            <div className={`flex-1 min-w-0 flex flex-col ${msg.role === 'user' ? 'items-end' : 'items-start'}`}>
              {msg.blocks?.map((block, blockIndex) => (
                <MessageBlock
                  key={block.id}
                  block={block}
                  role={msg.role}
                  streaming={msg.status === 'streaming' && blockIndex === msg.blocks.length - 1}
                />
              ))}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {stopReason && (
        <p className="text-xs text-muted-foreground text-center mb-2">
          {StopReasonLabels[stopReason] || stopReason}
        </p>
      )}

      {error && (
        <p className="text-xs text-destructive text-center mb-2">{error}</p>
      )}

      {(cwd || usage || mode) && (
        <div className="mb-2 flex min-w-0 items-center gap-3 text-[11px] text-muted-foreground">
          {cwd && <span className="min-w-0 flex-1 truncate font-mono">{cwd}</span>}
          {mode && <span className="shrink-0">Mode: {mode}</span>}
          {usage && <span className="shrink-0">Context: {usage}</span>}
        </div>
      )}

      <div className="flex gap-2">
        <textarea
          ref={inputRef}
          value={input}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          placeholder="Type a message..."
          disabled={streaming}
          rows={1}
          className="flex min-h-9 max-h-36 flex-1 resize-none rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50"
        />
        <Button onClick={streaming ? handleCancel : handleSend} disabled={!streaming && !input.trim()}>
          {streaming ? <Square className="size-4" /> : <Send className="size-4" />}
        </Button>
      </div>
    </div>
  )
}

export default function Chat() {
  const [activeSession, setActiveSession] = useState(null)
  const [shouldLoad, setShouldLoad] = useState(false)
  const refreshRef = useRef(null)
  const { agents, selectedAgentId, connected, reconnect } = useAgent()

  const agentForSession = useCallback((session) => {
    if (!session?.agentId) return null
    return agents.find((agent) => agent.id === session.agentId) || null
  }, [agents])

  const ensureSessionAgent = useCallback(async (session) => {
    const agent = agentForSession(session)
    if (!agent) return
    if (agent.id !== selectedAgentId || !connected) {
      await reconnect(agent)
    }
  }, [agentForSession, connected, reconnect, selectedAgentId])

  const handleSelectSession = async (session) => {
    if (sessionKey(activeSession) === sessionKey(session)) return
    await ensureSessionAgent(session)
    setActiveSession(session)
    setShouldLoad(true)
  }

  const handleNewSession = (session) => {
    setActiveSession(session)
    setShouldLoad(false)
  }

  const handleLoadComplete = useCallback(() => {
    setShouldLoad(false)
  }, [])

  const handleSessionClosed = useCallback((closedKey) => {
    if (sessionKey(activeSession) === closedKey) {
      setActiveSession(null)
    }
  }, [activeSession])

  const handleSessionDeleted = useCallback((deletedKey) => {
    if (sessionKey(activeSession) === deletedKey) {
      setActiveSession(null)
    }
  }, [activeSession])

  const handleBeforeCreateSession = useCallback(async (agent) => {
    if (!agent) throw new Error('No enabled agent configured')
    await reconnect(agent)
    return agent
  }, [reconnect])

  return (
    <div className="flex h-full min-h-0 min-w-0 w-full overflow-hidden">
      <div className="w-64 shrink-0 border-r">
        <SessionList
          activeSessionId={sessionKey(activeSession)}
          onSelectSession={handleSelectSession}
          onNewSession={handleNewSession}
          onRefresh={refreshRef}
          agents={agents}
          currentAgentId={selectedAgentId}
          onBeforeCreateSession={handleBeforeCreateSession}
          onBeforeSessionAction={ensureSessionAgent}
          onSessionClosed={handleSessionClosed}
          onSessionDeleted={handleSessionDeleted}
        />
      </div>
      <div className="flex-1 flex flex-col min-w-0 min-h-0 p-4 overflow-hidden">
        {!connected ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <AlertTriangle className="size-10 mb-3 text-amber-500" />
            <p className="text-sm font-medium">Agent not connected</p>
            <p className="text-xs mt-1">Create a session or enable an agent in Settings → Agents</p>
          </div>
        ) : activeSession ? (
          <ChatWindow
            key={sessionKey(activeSession)}
            sessionId={sessionKey(activeSession)}
            session={activeSession}
            shouldLoad={shouldLoad}
            onLoadComplete={handleLoadComplete}
          />
        ) : (
          <div className="flex h-full flex-col items-center justify-center text-center">
            <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-sm">
              <Bot className="size-8" />
            </div>
            <h1 className="text-2xl font-semibold tracking-normal text-foreground">
              Welcome to Miya
            </h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Select or create a session to begin.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
