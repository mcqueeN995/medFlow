package sm2

import (
	"math"
	"testing"
	"time"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 0.005 }

func TestApply_SequenceOfEasyGrades(t *testing.T) {
	p := DefaultProgress()

	p = Apply(p, 3)
	if !almostEqual(p.EaseFactor, 2.6) || p.IntervalDays != 1 || p.Repetitions != 1 {
		t.Fatalf("after 1st grade=3: %+v, want ease~2.6 interval=1 reps=1", p)
	}

	p = Apply(p, 3)
	if !almostEqual(p.EaseFactor, 2.7) || p.IntervalDays != 6 || p.Repetitions != 2 {
		t.Fatalf("after 2nd grade=3: %+v, want ease~2.7 interval=6 reps=2", p)
	}

	p = Apply(p, 3)
	if !almostEqual(p.EaseFactor, 2.8) || p.IntervalDays != 16 || p.Repetitions != 3 {
		t.Fatalf("after 3rd grade=3: %+v, want ease~2.8 interval=16 reps=3", p)
	}
}

func TestApply_GradeZeroResetsRepetitions(t *testing.T) {
	p := DefaultProgress()
	p = Apply(p, 0)
	if !almostEqual(p.EaseFactor, 1.7) || p.IntervalDays != 1 || p.Repetitions != 0 {
		t.Fatalf("Apply(default, 0) = %+v, want ease~1.7 interval=1 reps=0", p)
	}
}

func TestApply_GradeOne(t *testing.T) {
	p := DefaultProgress()
	p = Apply(p, 1)
	if !almostEqual(p.EaseFactor, 2.11) || p.IntervalDays != 1 || p.Repetitions != 0 {
		t.Fatalf("Apply(default, 1) = %+v, want ease~2.11 interval=1 reps=0", p)
	}
}

func TestApply_GradeTwo(t *testing.T) {
	p := DefaultProgress()
	p = Apply(p, 2)
	if !almostEqual(p.EaseFactor, 2.41) || p.IntervalDays != 1 || p.Repetitions != 1 {
		t.Fatalf("Apply(default, 2) = %+v, want ease~2.41 interval=1 reps=1", p)
	}
}

func TestApply_EaseFactorFloor(t *testing.T) {
	p := Progress{EaseFactor: 1.3, IntervalDays: 1, Repetitions: 0}
	for i := 0; i < 5; i++ {
		p = Apply(p, 0)
	}
	if p.EaseFactor < 1.3 {
		t.Fatalf("EaseFactor = %v, want >= 1.3 (floor)", p.EaseFactor)
	}
}

func TestApply_FailingAfterRepetitionsResetsToOneDay(t *testing.T) {
	p := DefaultProgress()
	p = Apply(p, 3)
	p = Apply(p, 3) // reps=2, interval=6
	p = Apply(p, 0) // forgot - resets
	if p.Repetitions != 0 || p.IntervalDays != 1 {
		t.Fatalf("Apply after forgetting = %+v, want reps=0 interval=1", p)
	}
}

func TestNextReviewDate(t *testing.T) {
	from := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	got := NextReviewDate(6, from)
	want := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("NextReviewDate(6, %v) = %v, want %v", from, got, want)
	}
}
