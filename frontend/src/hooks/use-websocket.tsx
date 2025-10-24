'use client'

import { useEffect, useRef, useState, useCallback } from 'react'
import { useAuth } from './use-auth'
import { api } from '@/lib/axios'

interface Message {
  id: number
  message_id: string
  user_id: string
  username: string
  content: string
  timestamp: string
  deleted?: boolean
  deleted_by_admin?: boolean
  temp_id?: string
  status?: 'pending' | 'sent' | 'error'
}

interface WebSocketMessage {
  type: 'message' | 'ack' | 'error' | 'message_deleted' | 'session_expired'
  id?: number
  message_id?: string
  user_id?: string
  username?: string
  content?: string
  timestamp?: string
  error?: string
  temp_id?: string
  status?: string
  deleted?: boolean
  deleted_by_admin?: boolean
}

interface UseWebSocketReturn {
  messages: Message[]
  isConnected: boolean
  sendMessage: (content: string) => void
  deleteMessage: (messageId: string) => void
  loadOlderMessages: () => Promise<void>
  hasMore: boolean
  isLoadingMore: boolean
}

export function useWebSocket(): UseWebSocketReturn {
  const { user } = useAuth()
  const [messages, setMessages] = useState<Message[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const [isLoadingMore, setIsLoadingMore] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const connect = useCallback(() => {
    if (!user) return

    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/api/ws'
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('WebSocket connected')
      setIsConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const data: WebSocketMessage = JSON.parse(event.data)

        switch (data.type) {
          case 'message':
            setMessages((prev) => {
              if (prev.length === 0) {
                return [{
                  id: data.id!,
                  message_id: data.message_id!,
                  user_id: data.user_id!,
                  username: data.username!,
                  content: data.content!,
                  timestamp: data.timestamp!,
                  status: 'sent',
                  deleted: data.deleted || false,
                  deleted_by_admin: data.deleted_by_admin || false
                }]
              }

              const isDuplicate = prev.some(msg =>
                msg.user_id === data.user_id &&
                msg.content === data.content &&
                Math.abs(new Date(msg.timestamp).getTime() - new Date(data.timestamp!).getTime()) < 2000
              )

              if (isDuplicate) {
                return prev.map(msg =>
                  msg.user_id === data.user_id && msg.content === data.content && !msg.message_id.startsWith('temp-')
                    ? msg
                    : msg.temp_id && msg.content === data.content
                    ? { ...msg, message_id: data.message_id!, status: 'sent', temp_id: undefined }
                    : msg
                )
              }

              return [{
                id: data.id!,
                message_id: data.message_id!,
                user_id: data.user_id!,
                username: data.username!,
                content: data.content!,
                timestamp: data.timestamp!,
                status: 'sent',
                deleted: data.deleted || false,
                deleted_by_admin: data.deleted_by_admin || false
              }, ...prev]
            })
            break

          case 'ack':
            if (data.status === 'success') {
              setMessages((prev) => prev.map((msg) =>
                msg.temp_id === data.temp_id
                  ? { ...msg, message_id: data.message_id!, status: 'sent', temp_id: undefined }
                  : msg
              ))
            } else {
              setMessages((prev) => prev.map((msg) =>
                msg.temp_id === data.temp_id
                  ? { ...msg, status: 'error' }
                  : msg
              ))
            }
            break

          case 'message_deleted':
            setMessages((prev) => prev.map((msg) =>
              msg.message_id === data.message_id
                ? { ...msg, deleted: true, deleted_by_admin: data.deleted_by_admin }
                : msg
            ))
            break

          case 'session_expired':
            console.warn('Session expired:', data.error)
            ws.close()
            break

          case 'error':
            console.error('WebSocket error:', data.error)
            break
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error.type)
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
      setIsConnected(false)

      reconnectTimeoutRef.current = setTimeout(() => {
        console.log('Reconnecting...')
        connect()
      }, 3000)
    }

    wsRef.current = ws
  }, [user])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  const sendMessage = useCallback((content: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.error('WebSocket not connected')
      return
    }

    if (!content.trim()) return

    const tempId = `temp-${Date.now()}-${Math.random()}`

    setMessages((prev) => [{
      id: 0,
      message_id: tempId,
      temp_id: tempId,
      user_id: user!.id,
      username: user!.username,
      content: content.trim(),
      timestamp: new Date().toISOString(),
      status: 'pending'
    }, ...prev])

    wsRef.current.send(JSON.stringify({
      type: 'send_message',
      temp_id: tempId,
      content: content.trim()
    }))
  }, [user])

  const deleteMessage = useCallback((messageId: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.error('WebSocket not connected')
      return
    }

    wsRef.current.send(JSON.stringify({
      type: 'delete_message',
      message_id: messageId
    }))
  }, [])

  const loadOlderMessages = useCallback(async () => {
    if (isLoadingMore || !hasMore || messages.length === 0) return

    setIsLoadingMore(true)

    try {
      const oldestMessageId = messages[messages.length - 1].id
      const response = await api.get(`/messages/before/${oldestMessageId}`)

      const olderMessages: Message[] = response.data.messages || []
      const fetchedHasMore: boolean = response.data.has_more || false

      setMessages((prev) => [...prev, ...olderMessages])
      setHasMore(fetchedHasMore)
    } catch (error) {
      console.error('Failed to load older messages:', error)
    } finally {
      setIsLoadingMore(false)
    }
  }, [messages, isLoadingMore, hasMore])

  return {
    messages,
    isConnected,
    sendMessage,
    deleteMessage,
    loadOlderMessages,
    hasMore,
    isLoadingMore
  }
}
