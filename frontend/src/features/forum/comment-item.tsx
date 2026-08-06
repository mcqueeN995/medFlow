import { useState } from 'react'
import { EyeOff, Flame, Reply, ShieldX, Trash2, TriangleAlert } from 'lucide-react'
import type { Comment } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { timeAgo } from '@/lib/forum'

interface CommentItemProps {
  comment: Comment
  currentUserId?: string
  reacted: boolean
  onReact: (id: string) => void
  onReply: (parentId: string, content: string) => Promise<void>
  onDelete: (id: string) => void
  onReport: (id: string) => void
  canModerate?: boolean
  isAdmin?: boolean
  onHide?: (id: string) => void
  onAdminDelete?: (id: string) => void
  nested?: boolean
}

export function CommentItem({
  comment,
  currentUserId,
  reacted,
  onReact,
  onReply,
  onDelete,
  onReport,
  canModerate,
  isAdmin,
  onHide,
  onAdminDelete,
  nested,
}: CommentItemProps) {
  const [replying, setReplying] = useState(false)
  const [replyText, setReplyText] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const isOwner = comment.author?.id === currentUserId

  async function submitReply() {
    if (!replyText.trim()) return
    setSubmitting(true)
    try {
      await onReply(comment.id!, replyText.trim())
      setReplyText('')
      setReplying(false)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className={cn('flex flex-col gap-1.5', nested && 'ml-6 border-l border-border pl-4')}>
      <div className="flex items-center gap-2 text-xs">
        <span className="font-medium text-foreground">{comment.author?.nickname ?? 'Аноним'}</span>
        <span className="text-muted-foreground">{timeAgo(comment.created_at)}</span>
      </div>
      <p className="text-sm text-foreground">{comment.content}</p>
      <div className="flex items-center gap-3 text-xs text-muted-foreground">
        <button
          type="button"
          onClick={() => onReact(comment.id!)}
          className={cn('flex items-center gap-1 hover:text-foreground', reacted && 'text-accent')}
        >
          <Flame className="size-3.5" /> {comment.likes_count ?? 0}
        </button>
        <button type="button" onClick={() => setReplying((v) => !v)} className="flex items-center gap-1 hover:text-foreground">
          <Reply className="size-3.5" /> Ответить
        </button>
        {isOwner ? (
          <button type="button" onClick={() => onDelete(comment.id!)} className="flex items-center gap-1 hover:text-destructive">
            <Trash2 className="size-3.5" /> Удалить
          </button>
        ) : (
          <button type="button" onClick={() => onReport(comment.id!)} className="flex items-center gap-1 hover:text-destructive">
            <TriangleAlert className="size-3.5" /> Пожаловаться
          </button>
        )}
        {canModerate && (
          <>
            {!comment.hidden_at && (
              <button type="button" onClick={() => onHide?.(comment.id!)} className="flex items-center gap-1 hover:text-destructive">
                <EyeOff className="size-3.5" /> Скрыть
              </button>
            )}
            {isAdmin && (
              <button type="button" onClick={() => onAdminDelete?.(comment.id!)} className="flex items-center gap-1 hover:text-destructive">
                <ShieldX className="size-3.5" /> Удалить (админ)
              </button>
            )}
          </>
        )}
      </div>

      {replying && (
        <div className="mt-1 flex flex-col gap-2">
          <Textarea
            value={replyText}
            onChange={(e) => setReplyText(e.target.value)}
            placeholder="Ваш ответ…"
            className="min-h-16 rounded-xl text-sm"
            maxLength={5000}
          />
          <div className="flex gap-2">
            <Button size="sm" className="h-8 rounded-full" disabled={submitting} onClick={submitReply}>
              {submitting ? 'Отправка…' : 'Ответить'}
            </Button>
            <Button size="sm" variant="ghost" className="h-8 rounded-full" onClick={() => setReplying(false)}>
              Отмена
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
