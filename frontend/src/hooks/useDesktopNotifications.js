import { useEffect, useRef } from 'react'
import { Events } from '@wailsio/runtime'
import {
  CheckNotificationAuthorization,
  InitializeNotifications,
  IsNotificationAvailable,
  RequestNotificationAuthorization,
  SendNotification,
} from '../../wailsjs/runtime/runtime'

export const DESKTOP_NOTIFICATIONS_KEY = 'miya.desktopNotifications.enabled'
export const DESKTOP_NOTIFICATIONS_EVENT = 'miya:desktop-notifications-changed'

export function desktopNotificationsEnabled() {
  if (typeof window === 'undefined') return true
  return window.localStorage.getItem(DESKTOP_NOTIFICATIONS_KEY) !== 'false'
}

export function setDesktopNotificationsEnabled(enabled) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(DESKTOP_NOTIFICATIONS_KEY, enabled ? 'true' : 'false')
  window.dispatchEvent(new CustomEvent(DESKTOP_NOTIFICATIONS_EVENT, { detail: { enabled } }))
}

export default function useDesktopNotifications(navigate) {
  const enabledRef = useRef(desktopNotificationsEnabled())
  const appActiveRef = useRef(true)
  const initializedRef = useRef(false)
  const initializingRef = useRef(null)
  const messageStatusRef = useRef(new Map())
  const notifiedRef = useRef(new Set())

  useEffect(() => {
    const syncEnabled = () => {
      enabledRef.current = desktopNotificationsEnabled()
    }
    window.addEventListener(DESKTOP_NOTIFICATIONS_EVENT, syncEnabled)
    window.addEventListener('storage', syncEnabled)
    return () => {
      window.removeEventListener(DESKTOP_NOTIFICATIONS_EVENT, syncEnabled)
      window.removeEventListener('storage', syncEnabled)
    }
  }, [])

  useEffect(() => {
    const setActive = (active) => {
      appActiveRef.current = active && document.visibilityState !== 'hidden'
    }
    const onFocus = () => setActive(true)
    const onBlur = () => setActive(false)
    const onVisibility = () => {
      appActiveRef.current = document.visibilityState !== 'hidden' && document.hasFocus()
    }

    window.addEventListener('focus', onFocus)
    window.addEventListener('blur', onBlur)
    document.addEventListener('visibilitychange', onVisibility)
    const shownCleanup = Events.On('app:window-shown', () => setActive(true))
    const hiddenCleanup = Events.On('app:window-hidden', () => setActive(false))

    return () => {
      window.removeEventListener('focus', onFocus)
      window.removeEventListener('blur', onBlur)
      document.removeEventListener('visibilitychange', onVisibility)
      shownCleanup()
      hiddenCleanup()
    }
  }, [])

  useEffect(() => {
    const cleanup = Events.On('notification:action', () => {
      navigate?.('chat')
    })
    return () => cleanup()
  }, [navigate])

  useEffect(() => {
    const cleanup = Events.On('conversation:update', (event) => {
      const snapshot = event?.data ?? event
      const conversation = snapshot?.conversation
      if (!conversation?.messages?.length) return

      for (const message of conversation.messages) {
        const messageKey = `${conversation.id}:${message.id}`
        const previousStatus = messageStatusRef.current.get(messageKey)
        messageStatusRef.current.set(messageKey, message.status)

        if (message.role !== 'assistant') continue
        if (message.status !== 'complete' && message.status !== 'failed') continue
        if (previousStatus !== 'streaming') continue
        if (notifiedRef.current.has(messageKey)) continue

        notifiedRef.current.add(messageKey)
        if (!enabledRef.current || appActiveRef.current) continue

        const body = notificationBody(message)
        const title = conversation.title || agentTitle(conversation) || 'Miya Desktop'
        sendDesktopNotification({
          id: notificationID(messageKey),
          title,
          body: body || (message.status === 'failed' ? 'Agent response failed.' : 'Agent response completed.'),
          data: {
            conversationId: conversation.id,
            acpSessionId: conversation.acpSessionId,
            messageId: message.id,
          },
        }, initializedRef, initializingRef)
      }
    })

    return () => cleanup()
  }, [])
}

async function sendDesktopNotification(options, initializedRef, initializingRef) {
  try {
    const ready = await ensureNotificationsReady(initializedRef, initializingRef)
    if (!ready) return
    await SendNotification(options)
  } catch (err) {
    console.warn('Failed to send desktop notification', err)
  }
}

async function ensureNotificationsReady(initializedRef, initializingRef) {
  if (initializedRef.current) return true
  if (initializingRef.current) return initializingRef.current

  initializingRef.current = (async () => {
    const available = await IsNotificationAvailable()
    if (!available) return false
    await InitializeNotifications()
    let authorized = await CheckNotificationAuthorization()
    if (!authorized) {
      authorized = await RequestNotificationAuthorization()
    }
    initializedRef.current = Boolean(authorized)
    return initializedRef.current
  })()

  try {
    return await initializingRef.current
  } finally {
    initializingRef.current = null
  }
}

function notificationID(value) {
  return value.replace(/[^a-zA-Z0-9_.:-]/g, '-')
}

function agentTitle(conversation) {
  if (conversation?.source?.channel) return conversation.source.channel
  if (conversation?.runtimeId) return conversation.runtimeId
  return ''
}

function notificationBody(message) {
  const text = message.blocks
    ?.map((block) => {
      if (block.type === 'text' || block.type === 'markdown') return block.content
      if (block.type === 'image') return block.name || 'Image attachment'
      if (block.type === 'audio') return block.name || 'Audio attachment'
      if (block.type === 'resource') return block.name || 'File attachment'
      return ''
    })
    .filter(Boolean)
    .join('\n')
    .trim()

  if (!text) return ''
  return text.length > 180 ? `${text.slice(0, 177)}...` : text
}
