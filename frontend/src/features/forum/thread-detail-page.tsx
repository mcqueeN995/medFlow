import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Eye, EyeOff, Flame, MessageCircle, Pencil, Send, ShieldX, Trash2, TriangleAlert } from 'lucide-react'
import { toast } from 'sonner'
import {
  deleteAdminCommentsId,
  deleteAdminThreadsId,
  deleteCommentsId,
  deleteCommentsIdReactions,
  deleteCommentsIdVote,
  deleteThreadsId,
  deleteThreadsIdReactions,
  getThreadsId,
  getThreadsIdComments,
  patchThreadsId,
  postAdminCommentsIdHide,
  postAdminThreadsIdHide,
  postCommentsIdReactions,
  postCommentsIdReport,
  postCommentsIdVote,
  postThreadsThreadIdComments,
  postThreadsIdReactions,
  postThreadsIdReport,
} from '@/api/generated/medFlowAPI'
import { UserRole } from '@/api/generated'
import type { CommentTree, Thread } from '@/api/generated'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth-store'
import { cn } from '@/lib/utils'
import { THREAD_TAG_LABELS, timeAgo } from '@/lib/forum'
import { CommentItem } from './comment-item'

export function ThreadDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const currentUserId = useAuthStore((s) => s.user?.id)
  const role = useAuthStore((s) => s.user?.role)
  const canModerate = role === UserRole.moderator || role === UserRole.admin
  const isAdmin = role === UserRole.admin

  const [thread, setThread] = useState<Thread | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)

  const [comments, setComments] = useState<CommentTree[]>([])
  const [commentsLoading, setCommentsLoading] = useState(true)
  const [commentText, setCommentText] = useState('')
  const [postingComment, setPostingComment] = useState(false)
  const [sort, setSort] = useState<'new' | 'best'>('new')

  const [editing, setEditing] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [editContent, setEditContent] = useState('')

  // Реакцию "своя/чужая" API не отдаёт (Thread/Comment не хранят её в контракте) -
  // поэтому подсветка "я лайкнул" живёт только в рамках текущей сессии страницы.
  const [reactedThread, setReactedThread] = useState(false)
  const [reactedComments, setReactedComments] = useState<Set<string>>(new Set())

  function loadThread() {
    if (!id) return
    setLoading(true)
    setNotFound(false)
    getThreadsId(id)
      .then((t) => {
        setThread(t)
        setEditTitle(t.title ?? '')
        setEditContent(t.content ?? '')
      })
      .catch(() => setNotFound(true))
      .finally(() => setLoading(false))
  }

  function loadComments() {
    if (!id) return
    setCommentsLoading(true)
    getThreadsIdComments(id, { limit: 100, sort })
      .then((res) => setComments(res.data ?? []))
      .finally(() => setCommentsLoading(false))
  }

  useEffect(() => {
    loadThread()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  useEffect(() => {
    loadComments()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, sort])

  async function toggleThreadReaction() {
    if (!id) return
    try {
      if (reactedThread) {
        await deleteThreadsIdReactions(id, { emoji: '🔥' })
        setThread((t) => (t ? { ...t, likes_count: Math.max(0, (t.likes_count ?? 0) - 1) } : t))
      } else {
        await postThreadsIdReactions(id, { emoji: '🔥' })
        setThread((t) => (t ? { ...t, likes_count: (t.likes_count ?? 0) + 1 } : t))
      }
      setReactedThread((v) => !v)
    } catch {
      toast.error('Не удалось поставить реакцию')
    }
  }

  async function toggleCommentReaction(commentId: string) {
    const reacted = reactedComments.has(commentId)
    try {
      if (reacted) {
        // DELETE-эндпоинт реакции комментария требует query-параметр emoji в
        // контракте, но у пользователя максимум одна реакция на цель, так что
        // значение не влияет на результат - можно всегда слать тот же эмодзи.
        await deleteCommentsIdReactions(commentId, { emoji: '🔥' })
      } else {
        await postCommentsIdReactions(commentId, { emoji: '🔥' })
      }
      setReactedComments((prev) => {
        const next = new Set(prev)
        if (reacted) next.delete(commentId)
        else next.add(commentId)
        return next
      })
      setComments((prev) =>
        prev.map((c) => {
          const bump = (n?: number) => Math.max(0, (n ?? 0) + (reacted ? -1 : 1))
          if (c.id === commentId) return { ...c, likes_count: bump(c.likes_count) }
          if (c.replies?.some((r) => r.id === commentId)) {
            return { ...c, replies: c.replies.map((r) => (r.id === commentId ? { ...r, likes_count: bump(r.likes_count) } : r)) }
          }
          return c
        }),
      )
    } catch {
      toast.error('Не удалось поставить реакцию')
    }
  }

  function findComment(commentId: string): CommentTree | undefined {
    for (const c of comments) {
      if (c.id === commentId) return c
      const reply = c.replies?.find((r) => r.id === commentId)
      if (reply) return reply as CommentTree
    }
    return undefined
  }

  function applyVoteResult(commentId: string, score?: number, myVote?: 'up' | 'down' | null) {
    setComments((prev) =>
      prev.map((c) => {
        if (c.id === commentId) return { ...c, vote_score: score ?? 0, my_vote: myVote ?? undefined }
        if (c.replies?.some((r) => r.id === commentId)) {
          return {
            ...c,
            replies: c.replies.map((r) => (r.id === commentId ? { ...r, vote_score: score ?? 0, my_vote: myVote ?? undefined } : r)),
          }
        }
        return c
      }),
    )
  }

  // Клик по уже выбранному направлению снимает голос (toggle); клик по
  // противоположному - меняет направление одним запросом (backend делает
  // ON CONFLICT DO UPDATE, отдельного вызова "снять старый голос" не нужно).
  async function voteComment(commentId: string, direction: 'up' | 'down') {
    const current = findComment(commentId)
    try {
      const result =
        current?.my_vote === direction
          ? await deleteCommentsIdVote(commentId)
          : await postCommentsIdVote(commentId, { direction })
      applyVoteResult(commentId, result.score, result.my_vote)
    } catch {
      toast.error('Не удалось проголосовать')
    }
  }

  async function submitComment() {
    if (!id || !commentText.trim()) return
    setPostingComment(true)
    try {
      await postThreadsThreadIdComments(id, { content: commentText.trim() })
      setCommentText('')
      loadComments()
      setThread((t) => (t ? { ...t, comments_count: (t.comments_count ?? 0) + 1 } : t))
    } catch {
      toast.error('Не удалось отправить комментарий')
    } finally {
      setPostingComment(false)
    }
  }

  async function submitReply(parentId: string, content: string) {
    if (!id) return
    try {
      await postThreadsThreadIdComments(id, { content, parent_id: parentId })
      loadComments()
      setThread((t) => (t ? { ...t, comments_count: (t.comments_count ?? 0) + 1 } : t))
    } catch {
      toast.error('Не удалось отправить ответ')
    }
  }

  function deleteComment(commentId: string) {
    if (!window.confirm('Удалить комментарий?')) return
    deleteCommentsId(commentId)
      .then(() => {
        loadComments()
        setThread((t) => (t ? { ...t, comments_count: Math.max(0, (t.comments_count ?? 0) - 1) } : t))
      })
      .catch(() => toast.error('Не удалось удалить комментарий'))
  }

  async function saveEdit() {
    if (!id) return
    try {
      const updated = await patchThreadsId(id, { title: editTitle.trim(), content: editContent.trim() })
      setThread(updated)
      setEditing(false)
      toast.success('Тред обновлён')
    } catch {
      toast.error('Не удалось сохранить изменения')
    }
  }

  async function deleteThread() {
    if (!id) return
    if (!window.confirm('Удалить тред безвозвратно?')) return
    try {
      await deleteThreadsId(id)
      toast.success('Тред удалён')
      navigate('/forum')
    } catch {
      toast.error('Не удалось удалить тред')
    }
  }

  async function reportThread() {
    if (!id) return
    const reason = window.prompt('Опишите, что не так с этим тредом:')
    if (!reason?.trim()) return
    try {
      await postThreadsIdReport(id, { reason: reason.trim() })
      toast.success('Жалоба отправлена модераторам')
    } catch {
      toast.error('Не удалось отправить жалобу')
    }
  }

  async function reportComment(commentId: string) {
    const reason = window.prompt('Опишите, что не так с этим комментарием:')
    if (!reason?.trim()) return
    try {
      await postCommentsIdReport(commentId, { reason: reason.trim() })
      toast.success('Жалоба отправлена модераторам')
    } catch {
      toast.error('Не удалось отправить жалобу')
    }
  }

  // ==================== Модерация (moderator+/admin) ====================

  async function hideThread() {
    if (!id) return
    const reason = window.prompt('Причина скрытия треда:')
    if (!reason?.trim()) return
    try {
      const updated = await postAdminThreadsIdHide(id, { reason: reason.trim() })
      setThread(updated)
      toast.success('Тред скрыт')
    } catch {
      toast.error('Не удалось скрыть тред')
    }
  }

  async function adminDeleteThread() {
    if (!id) return
    if (!window.confirm('Удалить тред безвозвратно как модератор/админ?')) return
    try {
      await deleteAdminThreadsId(id)
      toast.success('Тред удалён')
      navigate('/forum')
    } catch {
      toast.error('Не удалось удалить тред')
    }
  }

  async function hideComment(commentId: string) {
    const reason = window.prompt('Причина скрытия комментария:')
    if (!reason?.trim()) return
    try {
      await postAdminCommentsIdHide(commentId, { reason: reason.trim() })
      toast.success('Комментарий скрыт')
      loadComments()
    } catch {
      toast.error('Не удалось скрыть комментарий')
    }
  }

  async function adminDeleteComment(commentId: string) {
    if (!window.confirm('Удалить комментарий безвозвратно как модератор/админ?')) return
    try {
      await deleteAdminCommentsId(commentId)
      toast.success('Комментарий удалён')
      loadComments()
      setThread((t) => (t ? { ...t, comments_count: Math.max(0, (t.comments_count ?? 0) - 1) } : t))
    } catch {
      toast.error('Не удалось удалить комментарий')
    }
  }

  if (loading) {
    return (
      <div className="mx-auto flex max-w-2xl flex-col gap-4 p-6">
        <Skeleton className="h-6 w-24" />
        <Skeleton className="h-64 rounded-2xl" />
      </div>
    )
  }

  if (notFound || !thread) {
    return (
      <div className="flex flex-col items-center gap-3 p-16 text-center">
        <p className="font-medium text-foreground">Тред не найден</p>
        <Link to="/forum" className="text-sm text-accent underline underline-offset-2">
          Вернуться к тредам
        </Link>
      </div>
    )
  }

  const isOwner = thread.author?.id === currentUserId

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <Link to="/forum" className="flex w-fit items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="size-4" /> К тредам
      </Link>

      <div className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-6">
        <div className="flex items-center justify-between gap-2 text-sm">
          <div className="flex items-center gap-2">
            <span className="font-medium text-foreground">{thread.author?.nickname ?? 'Аноним'}</span>
            <span className="text-xs text-muted-foreground">{timeAgo(thread.created_at)}</span>
          </div>
          {isOwner && !editing && (
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              <button type="button" onClick={() => setEditing(true)} className="flex items-center gap-1 hover:text-foreground">
                <Pencil className="size-3.5" /> Изменить
              </button>
              <button type="button" onClick={deleteThread} className="flex items-center gap-1 hover:text-destructive">
                <Trash2 className="size-3.5" /> Удалить
              </button>
            </div>
          )}
        </div>

        {editing ? (
          <div className="flex flex-col gap-3">
            <Input value={editTitle} onChange={(e) => setEditTitle(e.target.value)} className="h-10 rounded-xl" maxLength={500} />
            <Textarea value={editContent} onChange={(e) => setEditContent(e.target.value)} className="min-h-32 rounded-xl" maxLength={50000} />
            <div className="flex gap-2">
              <Button size="sm" className="h-9 rounded-full" onClick={saveEdit}>
                Сохранить
              </Button>
              <Button size="sm" variant="ghost" className="h-9 rounded-full" onClick={() => setEditing(false)}>
                Отмена
              </Button>
            </div>
          </div>
        ) : (
          <>
            <h1 className="text-xl font-bold text-foreground">{thread.title}</h1>
            <p className="whitespace-pre-wrap text-sm text-foreground">{thread.content}</p>
          </>
        )}

        {thread.tags && thread.tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {thread.tags.map((tag) => (
              <span key={tag} className="rounded-full bg-secondary px-2.5 py-0.5 text-xs text-secondary-foreground">
                {THREAD_TAG_LABELS[tag]}
              </span>
            ))}
          </div>
        )}

        <div className="mt-1 flex items-center gap-4 text-sm text-muted-foreground">
          <button
            type="button"
            onClick={toggleThreadReaction}
            className={cn('flex items-center gap-1.5 hover:text-foreground', reactedThread && 'text-accent')}
          >
            <Flame className="size-4" /> {thread.likes_count ?? 0}
          </button>
          <span className="flex items-center gap-1.5">
            <MessageCircle className="size-4" /> {thread.comments_count ?? 0}
          </span>
          <span className="flex items-center gap-1.5">
            <Eye className="size-4" /> {thread.views_count ?? 0}
          </span>
          {!isOwner && (
            <button
              type="button"
              onClick={reportThread}
              className={cn('flex items-center gap-1.5 hover:text-destructive', !canModerate && 'ml-auto')}
            >
              <TriangleAlert className="size-4" /> Пожаловаться
            </button>
          )}
          {canModerate && (
            <div className="ml-auto flex items-center gap-3">
              {!thread.hidden_at && (
                <button type="button" onClick={hideThread} className="flex items-center gap-1.5 hover:text-destructive">
                  <EyeOff className="size-4" /> Скрыть
                </button>
              )}
              {isAdmin && (
                <button type="button" onClick={adminDeleteThread} className="flex items-center gap-1.5 hover:text-destructive">
                  <ShieldX className="size-4" /> Удалить (админ)
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="flex flex-col gap-4 rounded-2xl border border-border bg-card p-6">
        <div className="flex items-center justify-between gap-2">
          <h2 className="font-semibold text-foreground">Комментарии ({thread.comments_count ?? 0})</h2>
          <div className="flex items-center gap-1 rounded-full bg-secondary p-0.5 text-xs">
            <button
              type="button"
              onClick={() => setSort('new')}
              className={cn(
                'rounded-full px-3 py-1 transition-colors',
                sort === 'new' ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground',
              )}
            >
              Новые
            </button>
            <button
              type="button"
              onClick={() => setSort('best')}
              className={cn(
                'rounded-full px-3 py-1 transition-colors',
                sort === 'best' ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground',
              )}
            >
              Лучшие
            </button>
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <Textarea
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            placeholder="Написать комментарий…"
            className="min-h-20 rounded-xl"
            maxLength={5000}
          />
          <Button
            size="sm"
            className="h-9 w-fit rounded-full bg-linear-to-r from-primary to-accent text-primary-foreground"
            disabled={postingComment || !commentText.trim()}
            onClick={submitComment}
          >
            <Send className="size-4" />
            {postingComment ? 'Отправка…' : 'Отправить'}
          </Button>
        </div>

        {commentsLoading ? (
          <div className="flex flex-col gap-3">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-16 rounded-xl" />
            ))}
          </div>
        ) : comments.length === 0 ? (
          <p className="text-sm text-muted-foreground">Комментариев пока нет — станьте первым</p>
        ) : (
          <div className="flex flex-col gap-4">
            {comments.map((c) => (
              <div key={c.id} className="flex flex-col gap-3">
                <CommentItem
                  comment={c}
                  currentUserId={currentUserId}
                  reacted={reactedComments.has(c.id!)}
                  onReact={toggleCommentReaction}
                  onVote={voteComment}
                  onReply={submitReply}
                  onDelete={deleteComment}
                  onReport={reportComment}
                  canModerate={canModerate}
                  isAdmin={isAdmin}
                  onHide={hideComment}
                  onAdminDelete={adminDeleteComment}
                />
                {c.replies?.map((r) => (
                  <CommentItem
                    key={r.id}
                    comment={r}
                    currentUserId={currentUserId}
                    reacted={reactedComments.has(r.id!)}
                    onReact={toggleCommentReaction}
                    onVote={voteComment}
                    onReply={submitReply}
                    onDelete={deleteComment}
                    onReport={reportComment}
                    canModerate={canModerate}
                    isAdmin={isAdmin}
                    onHide={hideComment}
                    onAdminDelete={adminDeleteComment}
                    nested
                  />
                ))}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
