import { beforeEach, describe, expect, it } from 'vitest'
import { enqueueReview, listPendingReviews, removePendingReview } from './review-queue'

describe('review-queue (IndexedDB-backed offline queue)', () => {
  beforeEach(async () => {
    for (const entry of await listPendingReviews()) {
      await removePendingReview(entry.id)
    }
  })

  it('starts empty', async () => {
    expect(await listPendingReviews()).toEqual([])
  })

  it('enqueues a review and lists it back', async () => {
    await enqueueReview('card-1', 2)
    const pending = await listPendingReviews()
    expect(pending).toHaveLength(1)
    expect(pending[0]).toMatchObject({ cardId: 'card-1', grade: 2 })
    expect(pending[0].id).toBeTruthy()
    expect(pending[0].queuedAt).toBeTruthy()
  })

  it('supports multiple queued entries', async () => {
    await enqueueReview('card-1', 0)
    await enqueueReview('card-2', 3)
    const pending = await listPendingReviews()
    expect(pending).toHaveLength(2)
  })

  it('removes an entry by id', async () => {
    await enqueueReview('card-1', 1)
    const [entry] = await listPendingReviews()
    await removePendingReview(entry.id)
    expect(await listPendingReviews()).toEqual([])
  })
})
