import { http, HttpResponse } from 'msw'
import type { PushPreferences, PushSubscriptionRequest } from '@/api/generated'

const API = '*/api/v1'

let subscriptions: Array<{ id: string; endpoint: string }> = []
let preferences: PushPreferences = {
  thread_reply: true,
  comment_reply: true,
  reaction: true,
  card_task_done: true,
  card_task_failed: true,
  moderation_action: true,
  system: true,
}

export const pushHandlers = [
  http.post(`${API}/push/subscribe`, async ({ request }) => {
    const body = (await request.json()) as PushSubscriptionRequest
    const id = crypto.randomUUID()
    subscriptions = subscriptions.filter((s) => s.endpoint !== body.endpoint)
    subscriptions.push({ id, endpoint: body.endpoint })
    return HttpResponse.json({ id }, { status: 201 })
  }),

  http.delete(`${API}/push/unsubscribe`, ({ request }) => {
    const endpoint = new URL(request.url).searchParams.get('endpoint')
    const before = subscriptions.length
    subscriptions = subscriptions.filter((s) => s.endpoint !== endpoint)
    if (subscriptions.length === before) {
      return HttpResponse.json({ error: { code: 'NOT_FOUND', message: 'push subscription not found' } }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  http.patch(`${API}/push/preferences`, async ({ request }) => {
    const body = (await request.json()) as PushPreferences
    preferences = { ...preferences, ...body }
    return HttpResponse.json(preferences)
  }),
]
