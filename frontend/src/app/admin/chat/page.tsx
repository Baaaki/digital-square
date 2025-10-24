'use client'

import { useState, useRef, useEffect } from 'react'
import { useAuth } from '@/hooks/use-auth'
import { useWebSocket } from '@/hooks/use-websocket'
import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/components/theme-toggle'
import { useRouter } from 'next/navigation'

export default function AdminChatPage() {
  const router = useRouter()
  const { user, logout } = useAuth()
  const { messages, isConnected, sendMessage, deleteMessage, loadOlderMessages, hasMore, isLoadingMore } = useWebSocket()
  const [input, setInput] = useState('')
  const messagesContainerRef = useRef<HTMLDivElement>(null)

  // Redirect if not admin
  useEffect(() => {
    if (!user) {
      router.push('/login')
    } else if (user.role !== 'admin') {
      router.push('/chat')
    }
  }, [user, router])

  // Infinite scroll: Load older messages when scrolling to BOTTOM
  useEffect(() => {
    const container = messagesContainerRef.current
    if (!container) return

    const handleScroll = () => {
      const scrollBottom = container.scrollHeight - container.scrollTop - container.clientHeight

      if (scrollBottom < 100 && hasMore && !isLoadingMore) {
        const previousScrollTop = container.scrollTop

        loadOlderMessages().then(() => {
          requestAnimationFrame(() => {
            container.scrollTop = previousScrollTop
          })
        })
      }
    }

    container.addEventListener('scroll', handleScroll)
    return () => container.removeEventListener('scroll', handleScroll)
  }, [hasMore, isLoadingMore, loadOlderMessages])

  const handleSend = () => {
    if (input.trim()) {
      sendMessage(input)
      setInput('')
    }
  }

  if (!user || user.role !== 'admin') return null

  return (
    <div className="flex h-screen flex-col bg-zinc-50 dark:bg-zinc-950">
      {/* Navbar */}
      <nav className="sticky top-0 z-10 border-b bg-white/95 backdrop-blur-sm dark:bg-zinc-900/95">
        <div className="container mx-auto flex h-16 items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-blue-600 to-purple-600 text-white shadow-lg">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
            </div>
            <div>
              <h1 className="text-lg font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                Digital Square
              </h1>
              <div className="flex items-center gap-2 text-xs text-zinc-500">
                <span className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
                <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <ThemeToggle />
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/admin/users')}
            >
              👑 User Management
            </Button>
            <span className="rounded-full bg-yellow-100 px-3 py-1 text-xs font-semibold text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
              Admin
            </span>
            <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
              {user.username}
            </span>
            <Button variant="outline" size="sm" onClick={logout}>
              Logout
            </Button>
          </div>
        </div>
      </nav>

      {/* Input Box at Top (Twitter-style) */}
      <div className="border-b bg-zinc-50 dark:bg-zinc-950 p-4">
        <div className="container mx-auto max-w-2xl">
          <div className="bg-white dark:bg-zinc-900 rounded-xl border border-zinc-200 dark:border-zinc-800 p-4 shadow-sm">
            <div className="flex gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-yellow-500 to-orange-500 text-sm font-bold text-white">
                👑
              </div>
              <div className="flex-1 space-y-3">
                <textarea
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      handleSend()
                    }
                  }}
                  placeholder="What's happening?"
                  disabled={!isConnected}
                  className="w-full resize-none bg-transparent px-0 py-2 text-base outline-none placeholder:text-zinc-400 dark:text-zinc-100"
                  rows={2}
                />
                <div className="flex justify-end border-t border-zinc-100 dark:border-zinc-800 pt-3">
                  <Button
                    onClick={handleSend}
                    disabled={!isConnected || !input.trim()}
                    size="sm"
                    className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 font-semibold"
                  >
                    Post
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Messages (Twitter-style feed) - ADMIN VIEW */}
      <div ref={messagesContainerRef} className="flex-1 overflow-y-auto bg-zinc-50 dark:bg-zinc-950">
        <div className="container mx-auto max-w-2xl p-4 space-y-3">
          {messages.length === 0 && (
            <div className="flex h-64 items-center justify-center">
              <div className="text-center text-zinc-500">
                <svg className="mx-auto mb-4 h-16 w-16 opacity-20" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
                <p className="text-sm">No messages yet. Start the conversation!</p>
              </div>
            </div>
          )}

          {messages.map((msg) => {
            const isOwnMessage = msg.user_id === user.id

            // ADMIN FEATURE: See deleted message content
            if (msg.deleted) {
              return (
                <div key={msg.message_id} className="bg-red-50/50 dark:bg-red-950/20 rounded-xl border border-red-200 dark:border-red-900/50 p-4 shadow-sm">
                  <div className="flex gap-3">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-zinc-300 text-sm font-bold text-zinc-600 dark:bg-zinc-700 dark:text-zinc-400">
                      {msg.username?.charAt(0).toUpperCase() || '?'}
                    </div>
                    <div className="flex-1">
                      <div className="mb-1 flex items-center gap-2 flex-wrap">
                        <span className="font-semibold text-zinc-900 dark:text-zinc-100">
                          {msg.username}
                        </span>
                        <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/50 dark:text-red-300">
                          🗑️ {msg.deleted_by_admin ? 'Deleted by admin' : 'Deleted by user'}
                        </span>
                        <span className="text-sm text-zinc-400">·</span>
                        <span className="text-sm text-zinc-500 dark:text-zinc-400">
                          {new Date(msg.timestamp).toLocaleDateString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit'
                          })}
                        </span>
                      </div>
                      <p className="text-zinc-700 line-through dark:text-zinc-300 leading-relaxed">{msg.content}</p>
                      <p className="mt-2 text-xs text-red-600 dark:text-red-400 font-medium">
                        ℹ️ Only admins can see this content
                      </p>
                    </div>
                  </div>
                </div>
              )
            }

            // Normal message (Twitter-style post card)
            return (
              <div
                key={msg.message_id}
                className="bg-white dark:bg-zinc-900 rounded-xl border border-zinc-200 dark:border-zinc-800 p-4 shadow-sm transition-all hover:shadow-md hover:border-zinc-300 dark:hover:border-zinc-700 relative group"
              >
                {/* Delete button - top right (ADMIN can delete ANY message) */}
                <button
                  onClick={() => deleteMessage(msg.message_id)}
                  className="absolute top-3 right-3 p-1.5 rounded-lg text-zinc-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/20 dark:hover:text-red-400 transition-all opacity-0 group-hover:opacity-100 flex items-center gap-1"
                  title="Delete message (Admin)"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  <span className="text-[10px] font-medium">(Admin)</span>
                </button>

                <div className="flex gap-3">
                  {/* Avatar */}
                  <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-bold text-white ${
                    isOwnMessage
                      ? 'bg-gradient-to-br from-yellow-500 to-orange-500'
                      : 'bg-gradient-to-br from-zinc-600 to-zinc-700'
                  }`}>
                    {isOwnMessage ? '👑' : msg.username?.charAt(0).toUpperCase() || '?'}
                  </div>

                  {/* Content */}
                  <div className="flex-1 overflow-hidden">
                    {/* Header */}
                    <div className="mb-1 flex items-center gap-2 flex-wrap">
                      <span className="font-semibold text-zinc-900 dark:text-zinc-100">
                        {msg.username}
                      </span>
                      {isOwnMessage && (
                        <span className="rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400">
                          You (Admin)
                        </span>
                      )}
                      <span className="text-sm text-zinc-400">·</span>
                      <span className="text-sm text-zinc-500 dark:text-zinc-400">
                        {new Date(msg.timestamp).toLocaleDateString('en-US', {
                          month: 'short',
                          day: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit'
                        })}
                      </span>
                      {/* Status indicators */}
                      {msg.status === 'pending' && <span className="text-xs">⏳</span>}
                      {msg.status === 'sent' && <span className="text-xs">✅</span>}
                      {msg.status === 'error' && <span className="text-xs text-red-400">❌</span>}
                    </div>

                    {/* Message content */}
                    <p className="break-words text-zinc-900 dark:text-zinc-100 leading-relaxed">
                      {msg.content}
                    </p>
                  </div>
                </div>
              </div>
            )
          })}

          {/* Loading indicator */}
          {isLoadingMore && (
            <div className="flex justify-center py-4">
              <svg className="animate-spin h-6 w-6 text-blue-600" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
