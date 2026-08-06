import { openDB, type DBSchema } from 'idb'

interface PendingReview {
  id: string
  cardId: string
  grade: number
  queuedAt: string
}

interface ReviewQueueDB extends DBSchema {
  pending: {
    key: string
    value: PendingReview
  }
}

const dbPromise = openDB<ReviewQueueDB>('medflow-review-queue', 1, {
  upgrade(db) {
    db.createObjectStore('pending', { keyPath: 'id' })
  },
})

export async function enqueueReview(cardId: string, grade: number): Promise<void> {
  const db = await dbPromise
  const entry: PendingReview = { id: crypto.randomUUID(), cardId, grade, queuedAt: new Date().toISOString() }
  await db.put('pending', entry)
}

export async function listPendingReviews(): Promise<PendingReview[]> {
  const db = await dbPromise
  return db.getAll('pending')
}

export async function removePendingReview(id: string): Promise<void> {
  const db = await dbPromise
  await db.delete('pending', id)
}
