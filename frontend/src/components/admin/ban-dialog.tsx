'use client'

import { useState } from 'react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

interface BanDialogProps {
  user: { id: string; username: string; ids?: string[] } | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (userId: string | string[], reason: string) => void
}

export function BanDialog({ user, open, onOpenChange, onConfirm }: BanDialogProps) {
  const [reason, setReason] = useState('')

  const handleConfirm = () => {
    if (user?.ids) {
      // Bulk ban
      onConfirm(user.ids, reason)
    } else if (user?.id) {
      // Single ban
      onConfirm(user.id, reason)
    }
    setReason('') // Clear reason after confirm
  }

  const isBulk = user?.ids && user.ids.length > 0

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="text-red-600">
            ⚠️ Ban {isBulk ? `${user.ids?.length} Users` : `User: ${user?.username}`}
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-3">
            <p className="text-red-600 font-semibold text-base">Warning: This action is irreversible!</p>
            <ul className="list-disc pl-5 space-y-1 text-sm">
              <li><strong>ALL messages</strong> from {isBulk ? 'these users' : 'this user'} will be deleted</li>
              <li><strong>IP address</strong> will be blocked</li>
              <li>{isBulk ? 'These users' : 'This user'} won&apos;t be able to log in again</li>
            </ul>
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="space-y-2">
          <Label htmlFor="reason">Reason (required)</Label>
          <Input
            id="reason"
            placeholder="e.g., spam, harassment, inappropriate content"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            autoFocus
          />
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => setReason('')}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={!reason.trim()}
            className="bg-red-600 hover:bg-red-700 disabled:opacity-50"
          >
            Confirm Ban
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
