import { useState, useEffect, useRef, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import SessionList from '@/components/SessionList'
import { useAgent } from '@/context/AgentContext'
import { SendPrompt, LoadSession } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime'
import {
  Send,
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
          <p key={i} className="truncate">{c.type === 'text' ? c.content : `${c.type} content`}</p>
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
  return (
    <div className="flex items-start gap-2 text-xs text-muted-foreground bg-muted/20 rounded-md p-2 my-1 italic border-l-2 border-muted-foreground/20 max-w-full">
      <Brain className="size-3 mt-0.5 shrink-0" />
      <span className="min-w-0 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">{text}</span>
    </div>
  )
}

function isNewMessage(event) {
  return event.type === 'user_message_chunk' || event.type === 'agent_message_chunk'
}

function appendChunk(msg, event) {
  const content = event.content?.content || ''
  const thought = event.content?.thought || ''
  return {
    ...msg,
    text: msg.text + content + thought,
  }
}

function closeStreaming(msg) {
  if (!msg.streaming) return msg
  return { ...msg, streaming: false }
}

// Message-boundary key: prefer the server-provided messageId. If it's
// missing (some agents omit it for streaming chunks), fall back to the
// role — same role + streaming means "keep appending".
function chunkKey(event) {
  return event.content?.messageId || null
}

function ChatWindow({ sessionId, session, shouldLoad, onLoadComplete }) {
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [stopReason, setStopReason] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const messagesEndRef = useRef(null)
  const loadedRef = useRef(false)

  useEffect(() => {
    if (shouldLoad && !loadedRef.current) {
      loadedRef.current = true
      setLoading(true)
      setMessages([])
      LoadSession(sessionId, session.cwd || '/tmp')
        .then(() => {
          setMessages((prev) => prev.map(closeStreaming))
        })
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
  }, [messages, scrollToBottom])

  useEffect(() => {
    const cleanup = EventsOn('session:update', (data) => {
      const { sessionId: sid, event } = data
      if (sid !== sessionId) return

      setMessages((prev) => {
        const msgs = [...prev]

        if (isNewMessage(event)) {
          const eventRole = event.type === 'user_message_chunk' ? 'user' : 'assistant'
          const eventKey = chunkKey(event)
          const last = msgs[msgs.length - 1]

          // Only append when we can prove it's the *same* message. If the
          // server supplies messageId, use it. Otherwise fall back to
          // "same role + still streaming" which catches live-typing chunks
          // but stops merging across replayed history messages that arrive
          // as one full chunk each.
          const sameMessage =
            last &&
            last.role === eventRole &&
            (eventKey ? last.messageId === eventKey : last.streaming && !last.messageId)

          if (sameMessage) {
            msgs[msgs.length - 1] = appendChunk(last, event)
            return msgs
          }

          if (last && last.streaming) {
            msgs[msgs.length - 1] = closeStreaming(last)
          }

          msgs.push({
            role: eventRole,
            text: event.content?.content || event.content?.thought || '',
            messageId: eventKey,
            streaming: true,
            tools: [],
            thoughts: [],
          })
        } else if (event.type === 'agent_thought_chunk') {
          const last = msgs[msgs.length - 1]
          if (last && last.role === 'assistant' && last.streaming) {
            msgs[msgs.length - 1] = {
              ...last,
              thoughts: [...(last.thoughts || []), event.content?.thought || ''],
            }
          }
        } else if (event.type === 'tool_call') {
          if (!event.tool) return msgs
          const last = msgs[msgs.length - 1]
          if (last && last.role === 'assistant') {
            msgs[msgs.length - 1] = {
              ...last,
              tools: [...(last.tools || []), event.tool],
            }
          }
        } else if (event.type === 'tool_call_update') {
          if (!event.tool) return msgs
          const last = msgs[msgs.length - 1]
          if (last && last.tools) {
            msgs[msgs.length - 1] = {
              ...last,
              tools: last.tools.map((t) =>
                t.toolCallId === event.tool?.toolCallId ? { ...t, ...event.tool } : t
              ),
            }
          }
        } else if (event.type === 'plan') {
          const last = msgs[msgs.length - 1]
          if (last && last.role === 'assistant') {
            msgs[msgs.length - 1] = { ...last, plan: event.plan }
          }
        }

        return msgs
      })
    })

    return () => {
      cleanup()
    }
  }, [sessionId])

  const handleSend = async () => {
    const text = input.trim()
    if (!text || streaming) return

    setInput('')
    setStreaming(true)
    setStopReason(null)
    setError(null)

    try {
      await SendPrompt(sessionId, text)
    } catch (err) {
      setError(err.toString())
    } finally {
      setStreaming(false)
      setMessages((prev) => prev.map(closeStreaming))
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

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

        {messages.map((msg, i) => (
          <div key={i} className={`flex gap-3 min-w-0 ${msg.role === 'user' ? 'flex-row-reverse' : ''}`}>
            <div
              className={`flex size-8 shrink-0 items-center justify-center rounded-full ${
                msg.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
              }`}
            >
              {msg.role === 'user' ? <User className="size-4" /> : <Bot className="size-4" />}
            </div>

            <div className={`flex-1 min-w-0 flex flex-col ${msg.role === 'user' ? 'items-end' : 'items-start'}`}>
              <div
                className={`rounded-lg px-3 py-2 text-sm whitespace-pre-wrap break-words [overflow-wrap:anywhere] max-w-[85%] ${
                  msg.role === 'user'
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted/50'
                }`}
              >
                {msg.text}
                {msg.streaming && <span className="inline-block w-1.5 h-4 bg-current ml-0.5 animate-pulse align-middle" />}
              </div>

              {msg.thoughts?.map((t, j) => (
                <ThoughtDisplay key={j} text={t} />
              ))}

              {msg.tools?.map((t, j) => (
                <ToolCallDisplay key={j} tool={t} />
              ))}

              {msg.plan?.entries?.length > 0 && (
                <div className="text-xs text-muted-foreground bg-muted/20 rounded-md p-2 my-1 space-y-1 max-w-full">
                  {msg.plan.entries.map((e, j) => (
                    <div key={j} className="flex items-start gap-2 min-w-0">
                      <span className={`size-1.5 mt-1.5 shrink-0 rounded-full ${
                        e.status === 'completed' ? 'bg-green-500' :
                        e.status === 'in_progress' ? 'bg-blue-500 animate-pulse' :
                        'bg-muted-foreground/30'
                      }`} />
                      <span className={`min-w-0 break-words [overflow-wrap:anywhere] ${e.status === 'completed' ? 'line-through opacity-60' : ''}`}>
                        {e.content}
                      </span>
                    </div>
                  ))}
                </div>
              )}
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

      <div className="flex gap-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type a message..."
          disabled={streaming}
          className="flex-1"
        />
        <Button onClick={handleSend} disabled={!input.trim() || streaming}>
          {streaming ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
        </Button>
      </div>
    </div>
  )
}

export default function Chat() {
  const [activeSession, setActiveSession] = useState(null)
  const [shouldLoad, setShouldLoad] = useState(false)
  const refreshRef = useRef(null)
  const { connected } = useAgent()

  const handleSelectSession = (session) => {
    if (activeSession?.id === session.id) return
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

  return (
    <div className="flex h-full min-h-0 min-w-0 w-full overflow-hidden">
      <div className="w-64 shrink-0 border-r">
        <SessionList
          activeSessionId={activeSession?.id}
          onSelectSession={handleSelectSession}
          onNewSession={handleNewSession}
          onRefresh={refreshRef}
          connected={connected}
        />
      </div>
      <div className="flex-1 flex flex-col min-w-0 min-h-0 p-4 overflow-hidden">
        {!connected ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <AlertTriangle className="size-10 mb-3 text-amber-500" />
            <p className="text-sm font-medium">Agent not connected</p>
            <p className="text-xs mt-1">Go to Settings → General to select and connect an agent</p>
          </div>
        ) : activeSession ? (
          <ChatWindow
            key={activeSession.id}
            sessionId={activeSession.id}
            session={activeSession}
            shouldLoad={shouldLoad}
            onLoadComplete={handleLoadComplete}
          />
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Bot className="size-10 mb-3" />
            <p className="text-sm">Select or create a session to start chatting</p>
          </div>
        )}
      </div>
    </div>
  )
}
