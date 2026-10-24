package service

import (
	"math"
	"testing"

	"weathertracker/internal/domain"
)

func TestRangeScore(t *testing.T) {
	ideal := 22.0
	tests := []struct {
		name    string
		value   float64
		min     float64
		max     float64
		ideal   *float64
		wantMin float64
		wantMax float64
	}{
		{name: "ideal value", value: 22, min: 18, max: 26, ideal: &ideal, wantMin: 100, wantMax: 100},
		{name: "inside band", value: 20, min: 18, max: 26, ideal: &ideal, wantMin: 70, wantMax: 100},
		{name: "at lower bound", value: 18, min: 18, max: 26, ideal: &ideal, wantMin: 70, wantMax: 100},
		{name: "below band degrades", value: 14, min: 18, max: 26, ideal: &ideal, wantMin: 0, wantMax: 70},
		{name: "above band degrades", value: 30, min: 18, max: 26, ideal: &ideal, wantMin: 0, wantMax: 70},
		{name: "no ideal, inside means full", value: 20, min: 18, max: 26, wantMin: 100, wantMax: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rangeScore(tt.value, tt.min, tt.max, tt.ideal)
			if got < tt.wantMin || got > tt.wantMax {
				t.Fatalf("rangeScore(%v) = %v, want between %v and %v", tt.value, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPrecipitationScore(t *testing.T) {
	tests := []struct {
		code      int
		wantValue float64
		wantScore float64
	}{
		{code: 0, wantValue: 0, wantScore: 100},
		{code: 61, wantValue: 80, wantScore: 20},
		{code: 95, wantValue: 95, wantScore: 5},
	}
	for _, tt := range tests {
		w := domain.CurrentWeather{WeatherCode: tt.code}
		if got := precipitationValue(w); got != tt.wantValue {
			t.Errorf("precipitationValue(code=%d) = %v, want %v", tt.code, got, tt.wantValue)
		}
		if got := precipitationScore(w); got != tt.wantScore {
			t.Errorf("precipitationScore(code=%d) = %v, want %v", tt.code, got, tt.wantScore)
		}
	}
}

func TestWeightedScore(t *testing.T) {
	criteria := domain.RankingCriteria{
		Temperature:   domain.MetricPreference{Enabled: true, Weight: 3},
		Precipitation: domain.PrecipitationCriterion{Enabled: true, Weight: 1},
	}
	metrics := []domain.MetricScore{
		{Metric: "temperature", Score: 100},
		{Metric: "precipitation", Score: 0},
	}
	// (100*3 + 0*1) / 4 = 75
	if got := weightedScore(metrics, criteria); got != 75 {
		t.Fatalf("weightedScore() = %v, want 75", got)
	}
	if got := weightedScore(nil, criteria); got != 0 {
		t.Fatalf("weightedScore(nil) = %v, want 0", got)
	}
}

func TestClassifyOrientation(t *testing.T) {
	shore := 90.0
	tests := []struct {
		name    string
		windDir float64
		shore   *float64
		want    domain.Orientation
	}{
		{name: "onshore", windDir: 90, shore: &shore, want: domain.OrientationOnshore},
		{name: "offshore", windDir: 270, shore: &shore, want: domain.OrientationOffshore},
		{name: "cross-shore", windDir: 0, shore: &shore, want: domain.OrientationCrossShore},
		{name: "unknown without shore normal", windDir: 90, want: domain.OrientationUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyOrientation(tt.windDir, tt.shore); got != tt.want {
				t.Fatalf("classifyOrientation(%v) = %s, want %s", tt.windDir, got, tt.want)
			}
		})
	}
}

func TestCircularMeanWrapsAcrossNorth(t *testing.T) {
	// 350° and 10° average to 0° (north), not 180°.
	if got := circularMean([]float64{350, 10}); math.Abs(got-0) > 0.5 && math.Abs(got-360) > 0.5 {
		t.Fatalf("circularMean() = %v, want ~0", got)
	}
}

func TestGustPenaltyIsHigherForBeginners(t *testing.T) {
	beginner := gustPenalty(domain.LevelBeginner, 8, 16)
	pro := gustPenalty(domain.LevelPro, 8, 16)
	if beginner <= pro {
		t.Fatalf("expected beginners to be penalised more: beginner=%v pro=%v", beginner, pro)
	}
	if gustPenalty(domain.LevelPro, 8, 8) != 0 {
		t.Fatal("no penalty expected when gusts do not exceed the average wind")
	}
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		in, fallback, lo, hi, want int
	}{
		{0, 3, 1, 7, 3},
		{9, 3, 1, 7, 7},
		{-2, 3, 1, 7, 1},
		{5, 3, 1, 7, 5},
	}
	for _, tt := range tests {
		if got := clampInt(tt.in, tt.fallback, tt.lo, tt.hi); got != tt.want {
			t.Errorf("clampInt(%d,%d,%d,%d) = %d, want %d", tt.in, tt.fallback, tt.lo, tt.hi, got, tt.want)
		}
	}
}
