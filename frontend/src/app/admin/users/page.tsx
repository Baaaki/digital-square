'use client'

import { useState, useEffect } from 'react'
import { useAuth } from '@/hooks/use-auth'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { BanDialog } from '@/components/admin/ban-dialog'
import { ThemeToggle } from '@/components/theme-toggle'
import api from '@/lib/axios'

interface User {
  id: string
  username: string
  email: string
  role: string
  deleted_at: string | null
}

export default function AdminUsersPage() {
  const router = useRouter()
  const { user: currentUser, logout } = useAuth()
  const [users, setUsers] = useState<User[]>([])
  const [selectedUsers, setSelectedUsers] = useState<string[]>([])
  const [banDialogOpen, setBanDialogOpen] = useState(false)
  const [userToBan, setUserToBan] = useState<{ id: string; username: string; ids?: string[] } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Redirect if not admin
  useEffect(() => {
    if (!currentUser) {
      router.push('/login')
    } else if (currentUser.role !== 'admin') {
      router.push('/chat')
    }
  }, [currentUser, router])

  // Fetch users
  useEffect(() => {
    if (currentUser?.role === 'admin') {
      fetchUsers()
    }
  }, [currentUser])

  const fetchUsers = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/users')
      setUsers(response.data.users || [])
      setError('')
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to fetch users')
      console.error('Error fetching users:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleBanSingle = (user: User) => {
    setUserToBan({ id: user.id, username: user.username })
    setBanDialogOpen(true)
  }

  const handleBanBulk = () => {
    const selectedUsernames = users
      .filter(u => selectedUsers.includes(u.id))
      .map(u => u.username)
      .join(', ')

    setUserToBan({
      id: '',
      username: `${selectedUsers.length} users (${selectedUsernames})`,
      ids: selectedUsers
    })
    setBanDialogOpen(true)
  }

  const confirmBan = async (userId: string | string[], reason: string) => {
    try {
      if (Array.isArray(userId)) {
        // Bulk ban
        await api.post('/admin/ban-bulk', {
          user_ids: userId,
          reason
        })
      } else {
        // Single ban
        await api.post('/admin/ban', {
          user_id: userId,
          reason
        })
      }

      setBanDialogOpen(false)
      setSelectedUsers([])
      setUserToBan(null)

      // Refresh user list
      await fetchUsers()
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to ban user(s)')
      console.error('Error banning user(s):', err)
    }
  }

  const toggleSelectAll = () => {
    if (selectedUsers.length === users.filter(u => !u.deleted_at).length) {
      setSelectedUsers([])
    } else {
      setSelectedUsers(users.filter(u => !u.deleted_at).map(u => u.id))
    }
  }

  const toggleSelectUser = (userId: string) => {
    setSelectedUsers(prev =>
      prev.includes(userId)
        ? prev.filter(id => id !== userId)
        : [...prev, userId]
    )
  }

  if (!currentUser || currentUser.role !== 'admin') {
    return null
  }

  return (
    <div className="flex h-screen flex-col bg-zinc-50 dark:bg-zinc-950">
      {/* Navbar */}
      <nav className="sticky top-0 z-10 border-b bg-white/95 backdrop-blur-sm dark:bg-zinc-900/95">
        <div className="container mx-auto flex h-16 items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-red-600 to-orange-600 text-white shadow-lg">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            </div>
            <div>
              <h1 className="text-lg font-bold bg-gradient-to-r from-red-600 to-orange-600 bg-clip-text text-transparent">
                Admin Panel
              </h1>
              <p className="text-xs text-zinc-500">User Management</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <ThemeToggle />
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/admin/chat')}
            >
              💬 Chat
            </Button>
            <span className="rounded-full bg-yellow-100 px-3 py-1 text-xs font-semibold text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
              👑 {currentUser.username}
            </span>
            <Button variant="outline" size="sm" onClick={logout}>
              Logout
            </Button>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <div className="flex-1 overflow-y-auto p-8">
        <div className="container mx-auto max-w-6xl">
          {/* Header */}
          <div className="mb-6 flex items-center justify-between">
            <div>
              <h2 className="text-3xl font-bold text-zinc-900 dark:text-zinc-100">
                User Management
              </h2>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Total users: {users.length} | Active: {users.filter(u => !u.deleted_at).length} | Banned: {users.filter(u => u.deleted_at).length}
              </p>
            </div>

            {selectedUsers.length > 0 && (
              <Button
                variant="destructive"
                onClick={handleBanBulk}
                className="bg-red-600 hover:bg-red-700"
              >
                🚫 Ban {selectedUsers.length} user{selectedUsers.length > 1 ? 's' : ''}
              </Button>
            )}
          </div>

          {/* Error Message */}
          {error && (
            <div className="mb-4 rounded-lg bg-red-100 p-4 text-red-700 dark:bg-red-900/30 dark:text-red-400">
              {error}
            </div>
          )}

          {/* Loading State */}
          {loading ? (
            <div className="flex h-64 items-center justify-center">
              <svg className="animate-spin h-8 w-8 text-blue-600" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>
          ) : (
            /* Users Table */
            <div className="rounded-lg border bg-white shadow dark:border-zinc-800 dark:bg-zinc-900">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">
                      <Checkbox
                        checked={selectedUsers.length === users.filter(u => !u.deleted_at).length && users.filter(u => !u.deleted_at).length > 0}
                        onCheckedChange={toggleSelectAll}
                      />
                    </TableHead>
                    <TableHead>Username</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center text-zinc-500">
                        No users found
                      </TableCell>
                    </TableRow>
                  ) : (
                    users.map(user => (
                      <TableRow key={user.id} className={user.deleted_at ? 'opacity-50' : ''}>
                        <TableCell>
                          {!user.deleted_at && (
                            <Checkbox
                              checked={selectedUsers.includes(user.id)}
                              onCheckedChange={() => toggleSelectUser(user.id)}
                            />
                          )}
                        </TableCell>
                        <TableCell className="font-medium">
                          {user.username}
                          {user.id === currentUser.id && (
                            <span className="ml-2 rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                              You
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="text-zinc-600 dark:text-zinc-400">
                          {user.email}
                        </TableCell>
                        <TableCell>
                          <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            user.role === 'admin'
                              ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
                              : 'bg-zinc-100 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-300'
                          }`}>
                            {user.role === 'admin' ? '👑 Admin' : 'User'}
                          </span>
                        </TableCell>
                        <TableCell>
                          {user.deleted_at ? (
                            <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400">
                              🚫 BANNED
                            </span>
                          ) : (
                            <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">
                              ✓ Active
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          {!user.deleted_at && user.id !== currentUser.id && (
                            <Button
                              variant="destructive"
                              size="sm"
                              onClick={() => handleBanSingle(user)}
                              className="bg-red-600 hover:bg-red-700"
                            >
                              Ban
                            </Button>
                          )}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      </div>

      {/* Ban Dialog */}
      <BanDialog
        user={userToBan}
        open={banDialogOpen}
        onOpenChange={setBanDialogOpen}
        onConfirm={confirmBan}
      />
    </div>
  )
}
