'use client'

import type { ReactNode } from 'react'

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-gradient-to-br from-blue-50 via-white to-purple-50 px-4 py-12 dark:from-zinc-950 dark:via-zinc-900 dark:to-purple-950">
      {/* Animated background blobs */}
      <div className="absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute -left-20 -top-20 h-96 w-96 animate-pulse rounded-full bg-blue-400/20 blur-3xl" />
        <div className="absolute -bottom-20 -right-20 h-96 w-96 animate-pulse rounded-full bg-purple-400/20 blur-3xl [animation-delay:1s]" />
      </div>

      {/* Logo/Branding */}
      <div className="absolute left-8 top-8 flex items-center gap-2">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-blue-600 to-purple-600 text-white shadow-lg">
          <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
          </svg>
        </div>
        <span className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          Digital Square
        </span>
      </div>

      {/* Main Content */}
      <div className="w-full">{children}</div>
    </div>
  )
}
