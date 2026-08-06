// Package sm2 реализует интервальное повторение по алгоритму SM-2. Это
// точный порт эталонной реализации фронтенда (frontend/src/lib/sm2.ts),
// который был написан первым и уже проверен в UI-моках — формулы обеих
// сторон обязаны совпадать 1:1, иначе прогресс повторения разъедется между
// dev (моки) и прод (этот пакет).
//
// ТЗ использует шкалу оценки 0-3 (0=не помню, 1=сложно, 2=норм, 3=легко),
// а классический SM-2 рассчитан на quality 0-5 - пересчитываем оценку в
// эквивалент 0-5 (grade * 5/3) перед применением стандартной формулы.
package sm2

import (
	"math"
	"time"
)

type Progress struct {
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
}

func DefaultProgress() Progress {
	return Progress{EaseFactor: 2.5, IntervalDays: 0, Repetitions: 0}
}

// Apply применяет оценку grade (0-3) к текущему прогрессу и возвращает новый.
func Apply(p Progress, grade int) Progress {
	quality := float64(grade) / 3 * 5
	repetitions, intervalDays, easeFactor := p.Repetitions, p.IntervalDays, p.EaseFactor

	if grade < 2 {
		repetitions = 0
		intervalDays = 1
	} else {
		repetitions++
		switch repetitions {
		case 1:
			intervalDays = 1
		case 2:
			intervalDays = 6
		default:
			intervalDays = int(math.Round(float64(intervalDays) * easeFactor))
		}
	}

	easeFactor = math.Max(1.3, easeFactor+(0.1-(5-quality)*(0.08+(5-quality)*0.02)))
	easeFactor = math.Round(easeFactor*100) / 100

	return Progress{EaseFactor: easeFactor, IntervalDays: intervalDays, Repetitions: repetitions}
}

// NextReviewDate - момент следующего повторения, intervalDays дней от from.
func NextReviewDate(intervalDays int, from time.Time) time.Time {
	return from.AddDate(0, 0, intervalDays)
}
