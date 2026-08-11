import { http, HttpResponse } from 'msw'
import { ReactionTargetType, ThreadTag } from '@/api/generated'
import type {
  Comment,
  CommentTree,
  CreateCommentRequest,
  CreateThreadRequest,
  PublicUser,
  Reaction,
  Report,
  Thread,
  ThreadListItem,
  UpdateThreadRequest,
  VoteRequest,
  VoteResult,
} from '@/api/generated'
import { users, userByToken, type MockUser } from './auth'

const API = '*/api/v1'

interface MockThread {
  id: string
  authorId: string
  title: string
  content: string
  tags: ThreadTag[]
  views_count: number
  likes_count: number
  comments_count: number
  deleted_at: string | null
  created_at: string
}

interface MockComment {
  id: string
  threadId: string
  parentId: string | null
  // replyToId - кому реально отвечали, если это не тот же комментарий, что и
  // parentId (после схлопывания дерева до 2 уровней) - см. forum_service.go.
  replyToId: string | null
  authorId: string
  content: string
  depth: number
  likes_count: number
  deleted_at: string | null
  hidden_at: string | null
  created_at: string
}

let threadCounter = 0
let commentCounter = 0

const threads: MockThread[] = [
  {
    id: 'thread-1',
    authorId: '11111111-1111-1111-1111-111111111111',
    title: 'Как готовиться к нормальной анатомии на 2 курсе?',
    content:
      'Всем привет! Скоро экзамен по анатомии, делюсь своим конспектом и хотел бы узнать, как вы готовитесь — по атласу Синельникова или больше по лекциям? Какие темы обычно самые сложные на устном?',
    tags: [ThreadTag.study, ThreadTag.help],
    views_count: 42,
    likes_count: 7,
    comments_count: 2,
    deleted_at: null,
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 26).toISOString(),
  },
  {
    id: 'thread-2',
    authorId: '22222222-2222-2222-2222-222222222222',
    title: 'Продам стетоскоп Littmann Classic III, почти новый',
    content: 'Использовал один семестр, состояние отличное. Цена договорная, забирайте в кампусе Сеченовки.',
    tags: [ThreadTag.marketplace],
    views_count: 18,
    likes_count: 1,
    comments_count: 0,
    deleted_at: null,
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 3).toISOString(),
  },
]
threadCounter = threads.length

const comments: MockComment[] = [
  {
    id: 'comment-1',
    threadId: 'thread-1',
    parentId: null,
    replyToId: null,
    authorId: '22222222-2222-2222-2222-222222222222',
    content: 'Я готовился по атласу + Пирогов.онлайн, устно обычно спрашивают топографию черепных нервов.',
    depth: 0,
    likes_count: 3,
    deleted_at: null,
    hidden_at: null,
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 20).toISOString(),
  },
  {
    id: 'comment-2',
    threadId: 'thread-1',
    parentId: 'comment-1',
    replyToId: null,
    authorId: '11111111-1111-1111-1111-111111111111',
    content: 'Спасибо! А препараты по остеологии показывали живьём или только на муляжах?',
    depth: 1,
    likes_count: 0,
    deleted_at: null,
    hidden_at: null,
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 18).toISOString(),
  },
]
commentCounter = comments.length

// user_id:target_type:target_id -> emoji. У пользователя максимум одна
// реакция на цель (см. uq_reactions_user_target в бэкенде) - повторный
// POST с другим emoji просто меняет эмодзи, не плодит вторую реакцию.
const reactions = new Map<string, string>()

function reactionKey(userId: string, targetType: ReactionTargetType, targetId: string) {
  return `${userId}:${targetType}:${targetId}`
}

// commentId -> userId -> направление голоса. Отдельная карта от reactions -
// голос (kind='vote' на бэкенде) сосуществует с эмодзи-реакцией на том же
// комментарии независимо, см. миграцию 000016_reactions_kind.
const commentVotes = new Map<string, Map<string, 'up' | 'down'>>()

function voteSummary(commentId: string, viewerId?: string): { score: number; myVote?: 'up' | 'down' } {
  const votes = commentVotes.get(commentId)
  if (!votes) return { score: 0 }
  let score = 0
  for (const direction of votes.values()) score += direction === 'up' ? 1 : -1
  return { score, myVote: viewerId ? votes.get(viewerId) : undefined }
}

function currentUser(request: Request): MockUser | undefined {
  return userByToken(request.headers.get('authorization'))
}

function unauthorized() {
  return HttpResponse.json({ error: { code: 'UNAUTHORIZED', message: 'authentication required' } }, { status: 401 })
}

function notFound(message = 'не найдено') {
  return HttpResponse.json({ error: { code: 'NOT_FOUND', message } }, { status: 404 })
}

function forbidden() {
  return HttpResponse.json({ error: { code: 'FORBIDDEN', message: 'вы не автор' } }, { status: 403 })
}

function toPublicUser(user: MockUser): PublicUser {
  return {
    id: user.id,
    nickname: user.nickname,
    university: user.university,
    course: user.course,
    faculty: user.faculty,
    created_at: user.created_at,
    threads_count: threads.filter((t) => t.authorId === user.id && !t.deleted_at).length,
  }
}

function authorOf(userId: string): PublicUser {
  const user = users.find((u) => u.id === userId)
  return user ? toPublicUser(user) : { id: userId, nickname: 'Удалённый пользователь' }
}

function toThread(t: MockThread): Thread {
  return {
    id: t.id,
    author: authorOf(t.authorId),
    title: t.title,
    content: t.content,
    tags: t.tags,
    views_count: t.views_count,
    likes_count: t.likes_count,
    comments_count: t.comments_count,
    deleted_at: t.deleted_at,
    created_at: t.created_at,
  }
}

function toThreadListItem(t: MockThread): ThreadListItem {
  const { content: _content, ...rest } = toThread(t)
  void _content
  return rest
}

// toComment - content уходит пустым для удалённых/скрытых комментариев,
// плашку-заглушку рендерит фронтенд по deleted_at/hidden_at (см.
// dto.ToComment на бэкенде - тот же принцип: реальный текст наружу не идёт).
function toComment(c: MockComment, viewerId?: string): Comment {
  const { score, myVote } = voteSummary(c.id, viewerId)
  const isRemoved = Boolean(c.deleted_at || c.hidden_at)
  return {
    id: c.id,
    author: authorOf(c.authorId),
    content: isRemoved ? '' : c.content,
    depth: c.depth,
    reply_to_id: c.replyToId,
    likes_count: c.likes_count,
    vote_score: score,
    my_vote: myVote,
    deleted_at: c.deleted_at,
    hidden_at: c.hidden_at,
    created_at: c.created_at,
  }
}

export const forumHandlers = [
  http.get(`${API}/threads`, ({ request }) => {
    const url = new URL(request.url)
    const tag = url.searchParams.get('tag')
    const authorId = url.searchParams.get('author_id')
    const q = url.searchParams.get('q')?.toLowerCase()
    const sort = url.searchParams.get('sort') ?? 'created_at_desc'
    const page = Number(url.searchParams.get('page') ?? '1')
    const limit = Number(url.searchParams.get('limit') ?? '20')

    let filtered = threads.filter((t) => !t.deleted_at)
    if (tag) filtered = filtered.filter((t) => t.tags.includes(tag as ThreadTag))
    if (authorId) filtered = filtered.filter((t) => t.authorId === authorId)
    if (q) filtered = filtered.filter((t) => t.title.toLowerCase().includes(q) || t.content.toLowerCase().includes(q))

    filtered = [...filtered].sort((a, b) =>
      sort === 'popular'
        ? b.likes_count + b.comments_count - (a.likes_count + a.comments_count)
        : b.created_at.localeCompare(a.created_at),
    )

    const total = filtered.length
    const start = (page - 1) * limit
    const page_data = filtered.slice(start, start + limit)

    return HttpResponse.json({
      data: page_data.map(toThreadListItem),
      pagination: { page, limit, total, has_next: start + limit < total },
    })
  }),

  http.post(`${API}/threads`, async ({ request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const body = (await request.json()) as CreateThreadRequest

    threadCounter += 1
    const thread: MockThread = {
      id: `thread-${threadCounter}`,
      authorId: user.id!,
      title: body.title,
      content: body.content,
      tags: body.tags ?? [],
      views_count: 0,
      likes_count: 0,
      comments_count: 0,
      deleted_at: null,
      created_at: new Date().toISOString(),
    }
    threads.unshift(thread)
    return HttpResponse.json(toThread(thread), { status: 201 })
  }),

  http.get(`${API}/threads/:id`, ({ params }) => {
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')
    thread.views_count += 1
    return HttpResponse.json(toThread(thread))
  }),

  http.patch(`${API}/threads/:id`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')
    if (thread.authorId !== user.id) return forbidden()

    const body = (await request.json()) as UpdateThreadRequest
    if (body.title !== undefined) thread.title = body.title
    if (body.content !== undefined) thread.content = body.content
    if (body.tags !== undefined) thread.tags = body.tags
    return HttpResponse.json(toThread(thread))
  }),

  http.delete(`${API}/threads/:id`, ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')
    if (thread.authorId !== user.id) return forbidden()

    thread.deleted_at = new Date().toISOString()
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/threads/:id/reactions`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')

    const { emoji } = (await request.json()) as { emoji: string }
    const key = reactionKey(user.id!, ReactionTargetType.thread, thread.id)
    if (!reactions.has(key)) thread.likes_count += 1
    reactions.set(key, emoji)

    const reaction: Reaction = {
      id: crypto.randomUUID(),
      emoji,
      target_type: ReactionTargetType.thread,
      target_id: thread.id,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(reaction, { status: 201 })
  }),

  http.delete(`${API}/threads/:id/reactions`, ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')

    const key = reactionKey(user.id!, ReactionTargetType.thread, thread.id)
    if (!reactions.has(key)) return notFound('реакция не найдена')
    reactions.delete(key)
    thread.likes_count = Math.max(0, thread.likes_count - 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/threads/:id/report`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')

    const { reason } = (await request.json()) as { reason: string }
    const report: Report = {
      id: crypto.randomUUID(),
      reporter_id: user.id!,
      target_type: 'thread',
      target_id: thread.id,
      reason,
      status: 'pending',
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(report, { status: 201 })
  }),

  http.get(`${API}/threads/:id/comments`, ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')

    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? '1')
    const limit = Number(url.searchParams.get('limit') ?? '50')
    const sort = url.searchParams.get('sort') ?? 'new'

    // Удалённые/скрытые комментарии намеренно остаются в выборке - иначе их
    // ответы осиротели бы визуально при исчезновении родителя (см. ту же
    // логику в CommentRepo.ListByThread на бэкенде).
    let topLevel = comments
      .filter((c) => c.threadId === thread.id && c.parentId === null)
      .sort((a, b) => a.created_at.localeCompare(b.created_at))
    if (sort === 'best') {
      topLevel = [...topLevel].sort((a, b) => voteSummary(b.id).score - voteSummary(a.id).score)
    }
    const total = topLevel.length
    const start = (page - 1) * limit
    const pageComments = topLevel.slice(start, start + limit)

    const data: CommentTree[] = pageComments.map((c) => ({
      ...toComment(c, user.id),
      replies: comments
        .filter((r) => r.parentId === c.id)
        .sort((a, b) => a.created_at.localeCompare(b.created_at))
        .map((r) => toComment(r, user.id)),
    }))

    return HttpResponse.json({ data, pagination: { page, limit, total, has_next: start + limit < total } })
  }),

  http.post(`${API}/threads/:id/comments`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const thread = threads.find((t) => t.id === params.id && !t.deleted_at)
    if (!thread) return notFound('тред не найден')

    const body = (await request.json()) as CreateCommentRequest

    let parentId: string | null = null
    let replyToId: string | null = null
    let depth = 0
    if (body.parent_id) {
      const parent = comments.find((c) => c.id === body.parent_id && c.threadId === thread.id && !c.deleted_at)
      if (!parent) {
        return HttpResponse.json(
          { error: { code: 'INVALID_PARENT', message: 'родительский комментарий не найден в этом треде' } },
          { status: 400 },
        )
      }
      // дерево ограничено 2 уровнями - ответ на ответ "схлопывается" к
      // родителю верхнего уровня, см. такую же логику в forum_service.go.
      // replyToId сохраняет исходного адресата, когда он отличается от
      // схлопнутого parentId - иначе теряется, кому реально отвечали.
      if (parent.depth === 0) {
        parentId = parent.id
      } else {
        parentId = parent.parentId
        replyToId = parent.id
      }
      depth = 1
    }

    commentCounter += 1
    const comment: MockComment = {
      id: `comment-${commentCounter}`,
      threadId: thread.id,
      parentId,
      replyToId,
      authorId: user.id!,
      content: body.content,
      depth,
      likes_count: 0,
      deleted_at: null,
      hidden_at: null,
      created_at: new Date().toISOString(),
    }
    comments.push(comment)
    thread.comments_count += 1

    return HttpResponse.json(toComment(comment), { status: 201 })
  }),

  http.patch(`${API}/comments/:id`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')
    if (comment.authorId !== user.id) return forbidden()

    const { content } = (await request.json()) as { content: string }
    comment.content = content
    return HttpResponse.json(toComment(comment))
  }),

  http.delete(`${API}/comments/:id`, ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')
    if (comment.authorId !== user.id) return forbidden()

    // comments_count не трогаем - комментарий остаётся плашкой-заглушкой в
    // дереве, слот никуда не девается (см. CommentRepo.SoftDelete на бэкенде).
    comment.deleted_at = new Date().toISOString()
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/comments/:id/reactions`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')

    const { emoji } = (await request.json()) as { emoji: string }
    const key = reactionKey(user.id!, ReactionTargetType.comment, comment.id)
    if (!reactions.has(key)) comment.likes_count += 1
    reactions.set(key, emoji)

    const reaction: Reaction = {
      id: crypto.randomUUID(),
      emoji,
      target_type: ReactionTargetType.comment,
      target_id: comment.id,
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(reaction, { status: 201 })
  }),

  http.delete(`${API}/comments/:id/reactions`, ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')

    const key = reactionKey(user.id!, ReactionTargetType.comment, comment.id)
    if (!reactions.has(key)) return notFound('реакция не найдена')
    reactions.delete(key)
    comment.likes_count = Math.max(0, comment.likes_count - 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post(`${API}/comments/:id/vote`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')

    const { direction } = (await request.json()) as VoteRequest
    let votes = commentVotes.get(comment.id)
    if (!votes) {
      votes = new Map()
      commentVotes.set(comment.id, votes)
    }
    votes.set(user.id!, direction as 'up' | 'down')

    const { score, myVote } = voteSummary(comment.id, user.id)
    const result: VoteResult = { score, my_vote: myVote }
    return HttpResponse.json(result)
  }),

  http.delete(`${API}/comments/:id/vote`, ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')

    const votes = commentVotes.get(comment.id)
    if (!votes?.has(user.id!)) return notFound('голос не найден')
    votes.delete(user.id!)

    const { score, myVote } = voteSummary(comment.id, user.id)
    const result: VoteResult = { score, my_vote: myVote }
    return HttpResponse.json(result)
  }),

  http.post(`${API}/comments/:id/report`, async ({ params, request }) => {
    const user = currentUser(request)
    if (!user) return unauthorized()
    const comment = comments.find((c) => c.id === params.id && !c.deleted_at)
    if (!comment) return notFound('комментарий не найден')

    const { reason } = (await request.json()) as { reason: string }
    const report: Report = {
      id: crypto.randomUUID(),
      reporter_id: user.id!,
      target_type: 'comment',
      target_id: comment.id,
      reason,
      status: 'pending',
      created_at: new Date().toISOString(),
    }
    return HttpResponse.json(report, { status: 201 })
  }),
]
