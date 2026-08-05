// Интервальное повторение по алгоритму SM-2 (см. Полная спецификация проекта).
// ТЗ использует шкалу оценки 0-3 (0=не помню, 1=сложно, 2=норм, 3=легко),
// а классический SM-2 рассчитан на quality 0-5 — пересчитываем оценку в
// эквивалент 0-5 (grade * 5/3) перед применением стандартной формулы.

export interface SM2Progress {
  easeFactor: number
  intervalDays: number
  repetitions: number
}

export const DEFAULT_SM2_PROGRESS: SM2Progress = { easeFactor: 2.5, intervalDays: 0, repetitions: 0 }

export const GRADE_LABELS: Record<number, string> = {
  0: 'Не помню',
  1: 'Сложно',
  2: 'Норм',
  3: 'Легко',
}

export function applySM2(progress: SM2Progress, grade: number): SM2Progress {
  const quality = (grade / 3) * 5
  let { repetitions, intervalDays } = progress
  let { easeFactor } = progress

  if (grade < 2) {
    repetitions = 0
    intervalDays = 1
  } else {
    repetitions += 1
    if (repetitions === 1) intervalDays = 1
    else if (repetitions === 2) intervalDays = 6
    else intervalDays = Math.round(intervalDays * easeFactor)
  }

  easeFactor = Math.max(1.3, easeFactor + (0.1 - (5 - quality) * (0.08 + (5 - quality) * 0.02)))

  return { easeFactor: Math.round(easeFactor * 100) / 100, intervalDays, repetitions }
}

export function nextReviewDate(intervalDays: number, from = new Date()): string {
  const d = new Date(from)
  d.setDate(d.getDate() + intervalDays)
  return d.toISOString()
}
