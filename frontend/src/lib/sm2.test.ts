import { describe, expect, it } from 'vitest'
import { applySM2, DEFAULT_SM2_PROGRESS, nextReviewDate } from './sm2'

describe('applySM2', () => {
  it('resets repetitions and interval to 1 day on a low grade (<2)', () => {
    const progress = { easeFactor: 2.5, intervalDays: 6, repetitions: 2 }
    const result = applySM2(progress, 0)
    expect(result.repetitions).toBe(0)
    expect(result.intervalDays).toBe(1)
  })

  it('first successful repetition sets interval to 1 day', () => {
    const result = applySM2(DEFAULT_SM2_PROGRESS, 3)
    expect(result.repetitions).toBe(1)
    expect(result.intervalDays).toBe(1)
  })

  it('second successful repetition sets interval to 6 days', () => {
    const afterFirst = applySM2(DEFAULT_SM2_PROGRESS, 3)
    const afterSecond = applySM2(afterFirst, 3)
    expect(afterSecond.repetitions).toBe(2)
    expect(afterSecond.intervalDays).toBe(6)
  })

  it('third+ successful repetition scales interval by easeFactor', () => {
    let progress = DEFAULT_SM2_PROGRESS
    progress = applySM2(progress, 3)
    progress = applySM2(progress, 3)
    const before = progress
    progress = applySM2(progress, 3)
    expect(progress.repetitions).toBe(3)
    expect(progress.intervalDays).toBe(Math.round(before.intervalDays * before.easeFactor))
  })

  it('never lets easeFactor drop below 1.3', () => {
    let progress = DEFAULT_SM2_PROGRESS
    for (let i = 0; i < 20; i++) {
      progress = applySM2(progress, 0)
    }
    expect(progress.easeFactor).toBeGreaterThanOrEqual(1.3)
  })
})

describe('nextReviewDate', () => {
  it('adds the given number of days to the reference date', () => {
    const from = new Date('2026-01-01T00:00:00.000Z')
    const result = nextReviewDate(6, from)
    expect(new Date(result).getUTCDate()).toBe(7)
  })

  it('returns an ISO string', () => {
    expect(nextReviewDate(1, new Date('2026-01-01T00:00:00.000Z'))).toMatch(/^\d{4}-\d{2}-\d{2}T/)
  })
})
