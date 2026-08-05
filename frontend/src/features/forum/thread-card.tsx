import { Link } from 'react-router-dom'
import { Eye, Flame, MessageCircle } from 'lucide-react'
import type { ThreadListItem } from '@/api/generated'
import { THREAD_TAG_LABELS, timeAgo } from '@/lib/forum'

export function ThreadCard({ thread }: { thread: ThreadListItem }) {
  return (
    <Link
      to={`/forum/${thread.id}`}
      className="flex flex-col gap-2.5 rounded-2xl border border-border bg-card p-5 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{thread.author?.nickname ?? 'Аноним'}</span>
        <span>{timeAgo(thread.created_at)}</span>
      </div>

      <h3 className="line-clamp-2 font-semibold text-foreground">{thread.title}</h3>

      {thread.tags && thread.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {thread.tags.map((tag) => (
            <span key={tag} className="rounded-full bg-secondary px-2.5 py-0.5 text-xs text-secondary-foreground">
              {THREAD_TAG_LABELS[tag]}
            </span>
          ))}
        </div>
      )}

      <div className="mt-1 flex items-center gap-4 text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <Flame className="size-3.5" /> {thread.likes_count ?? 0}
        </span>
        <span className="flex items-center gap-1">
          <MessageCircle className="size-3.5" /> {thread.comments_count ?? 0}
        </span>
        <span className="flex items-center gap-1">
          <Eye className="size-3.5" /> {thread.views_count ?? 0}
        </span>
      </div>
    </Link>
  )
}
