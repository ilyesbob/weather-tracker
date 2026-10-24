package service

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository"
)

// Wind defaults keep the service usable even before the rider saves a profile.
const (
	defaultWindMin = 5.0
	defaultWindMax = 12.0
	defaultDays    = 3
	defaultSlotHrs = 3
	maxDays        = 7
	maxSlotHrs     = 6
	maxResults     = 5
)

// User-facing messages, extracted so wording stays consistent and testable.
const (
	msgNoData       = "No forecast data is available for the selected spots."
	msgNoExactMatch = "No slot fully matches the requested wind range; showing the closest available options."
)

// Recommendation score weights: wind speed dominates, orientation refines it.
const (
	windScoreWeight        = 0.6
	orientationScoreWeight = 0.4
)

// Orientation thresholds expressed as angular distance (degrees) between the
// wind bearing and the seaward shore normal.
const (
	onshoreMaxDelta  = 45.0
	offshoreMinDelta = 135.0
)

// gustWarningRatio is the gust/average ratio above which the rider is warned.
const gustWarningRatio = 1.5

// Horizontal score constants used when ranking spots and rider levels.
const (
	orientationScoreOnshore       = 100.0
	orientationScoreCrossSkill    = 90.0
	orientationScoreCrossBeginner = 75.0
	orientationScoreOffshoreBegin = 15.0
	orientationScoreOffshoreInt   = 45.0
	orientationScoreOffshoreAdv   = 70.0
	orientationScoreOffshorePro   = 85.0
	orientationScoreUnknown       = 70.0
)

// Gust penalty shaping constants per skill level (scale, cap).
const (
	gustPenaltyScaleBeginner     = 30.0
	gustPenaltyScaleIntermediate = 20.0
	gustPenaltyScaleAdvanced     = 10.0
	gustPenaltyCapBeginner       = 25.0
	gustPenaltyCapIntermediate   = 15.0
	gustPenaltyCapAdvanced       = 10.0
)

// WindsurfService implements User Story 2: it analyses the forecast across
// several spots and recommends the best location + time slot for the rider's
// level, explaining the choice.
type WindsurfService struct {
	profiles       repository.WindsurfProfileRepository
	saved          repository.SavedLocationRepository
	catalog        repository.LocationRepository
	weather        repository.WeatherProvider
	staleThreshold time.Duration
	now            func() time.Time
}

// NewWindsurfService wires the windsurf service dependencies.
func NewWindsurfService(
	profiles repository.WindsurfProfileRepository,
	saved repository.SavedLocationRepository,
	catalog repository.LocationRepository,
	weather repository.WeatherProvider,
	staleThreshold time.Duration,
) *WindsurfService {
	return &WindsurfService{
		profiles:       profiles,
		saved:          saved,
		catalog:        catalog,
		weather:        weather,
		staleThreshold: staleThreshold,
		now:            time.Now,
	}
}

// ProfileInput is the rider profile request body.
type ProfileInput struct {
	Level        domain.SkillLevel `json:"level"`
	WindSpeedMin float64           `json:"windSpeedMin"`
	WindSpeedMax float64           `json:"windSpeedMax"`
}

// RecommendInput is the recommendation request body.
type RecommendInput struct {
	Level        *domain.SkillLevel `json:"level,omitempty"`
	WindSpeedMin *float64           `json:"windSpeedMin,omitempty"`
	WindSpeedMax *float64           `json:"windSpeedMax,omitempty"`
	Days         int                `json:"days,omitempty"`
	SlotHours    int                `json:"slotHours,omitempty"`
	Spots        []domain.Spot      `json:"spots,omitempty"`
}

// SaveProfile validates and persists the rider profile.
func (s *WindsurfService) SaveProfile(ctx context.Context, userID string, in ProfileInput) (domain.WindsurfProfile, error) {
	profile := domain.WindsurfProfile{
		UserID:       userID,
		Level:        in.Level,
		WindSpeedMin: in.WindSpeedMin,
		WindSpeedMax: in.WindSpeedMax,
		UpdatedAt:    s.now().UTC(),
	}
	if err := validateProfile(&profile); err != nil {
		return domain.WindsurfProfile{}, err
	}
	saved, err := s.profiles.Save(ctx, profile)
	if err != nil {
		return domain.WindsurfProfile{}, fmt.Errorf("save windsurf profile: %w", err)
	}
	return saved, nil
}

// Profile returns the stored rider profile.
func (s *WindsurfService) Profile(ctx context.Context, userID string) (domain.WindsurfProfile, error) {
	profile, err := s.profiles.ByUserID(ctx, userID)
	if err != nil {
		return domain.WindsurfProfile{}, err
	}
	return profile, nil
}

// Recommend analyses the forecast and returns ranked spot + time slot
// recommendations. When nothing matches the requested wind range the closest
// option is still returned together with an explanatory message.
func (s *WindsurfService) Recommend(ctx context.Context, userID string, in RecommendInput) (domain.RecommendationResult, error) {
	profile := s.mergeProfile(ctx, userID, in)
	if err := validateProfile(&profile); err != nil {
		return domain.RecommendationResult{}, err
	}

	days := clampInt(in.Days, defaultDays, 1, maxDays)
	slotHours := clampInt(in.SlotHours, defaultSlotHrs, 1, maxSlotHrs)

	spots, err := s.resolveSpots(ctx, userID, in.Spots)
	if err != nil {
		return domain.RecommendationResult{}, err
	}

	now := s.now().UTC()
	recommendations, anyAcceptable := s.collectRecommendations(ctx, spots, profile, days, slotHours, now)
	if len(recommendations) == 0 {
		return domain.RecommendationResult{
			GeneratedAt: now,
			Profile:     profile,
			Message:     msgNoData,
		}, nil
	}

	result := domain.RecommendationResult{
		GeneratedAt:     now,
		Profile:         profile,
		Recommendations: topRecommendations(recommendations),
	}
	if !anyAcceptable {
		result.Message = msgNoExactMatch
	}
	return result, nil
}

// collectRecommendations fetches each spot's forecast and keeps the best slot.
// A spot whose upstream call fails is skipped rather than failing the whole
// request, so one bad location cannot take down the comparison.
func (s *WindsurfService) collectRecommendations(ctx context.Context, spots []domain.Spot, profile domain.WindsurfProfile, days, slotHours int, now time.Time) ([]domain.Recommendation, bool) {
	recommendations := make([]domain.Recommendation, 0, len(spots))
	var anyAcceptable bool
	for _, spot := range spots {
		snapshot, err := s.weather.Fetch(ctx, *spot.Latitude, *spot.Longitude, days)
		if err != nil {
			continue // unavailable spot, not fatal
		}
		rec, ok := bestWindow(spot, snapshot, profile, slotHours, now, s.staleThreshold)
		if !ok {
			continue
		}
		anyAcceptable = anyAcceptable || rec.acceptable
		recommendations = append(recommendations, rec.recommendation)
	}
	return recommendations, anyAcceptable
}

// topRecommendations sorts by descending score, keeps the leading results and
// flags the winner. Ties resolve by start time for a stable UI.
func topRecommendations(recommendations []domain.Recommendation) []domain.Recommendation {
	slices.SortFunc(recommendations, func(a, b domain.Recommendation) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		return a.Start.Compare(b.Start)
	})
	if len(recommendations) > maxResults {
		recommendations = recommendations[:maxResults]
	}
	recommendations[0].IsBest = true
	return recommendations
}

func (s *WindsurfService) mergeProfile(ctx context.Context, userID string, in RecommendInput) domain.WindsurfProfile {
	profile := domain.WindsurfProfile{
		UserID:       userID,
		Level:        domain.LevelIntermediate,
		WindSpeedMin: defaultWindMin,
		WindSpeedMax: defaultWindMax,
	}
	if stored, err := s.profiles.ByUserID(ctx, userID); err == nil {
		profile = stored
	}
	if in.Level != nil {
		profile.Level = *in.Level
	}
	if in.WindSpeedMin != nil {
		profile.WindSpeedMin = *in.WindSpeedMin
	}
	if in.WindSpeedMax != nil {
		profile.WindSpeedMax = *in.WindSpeedMax
	}
	return profile
}

func (s *WindsurfService) resolveSpots(ctx context.Context, userID string, spots []domain.Spot) ([]domain.Spot, error) {
	if len(spots) == 0 {
		saved, err := s.saved.List(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("load saved locations: %w", err)
		}
		out := make([]domain.Spot, 0, len(saved))
		for _, sl := range saved {
			out = append(out, domain.Spot{
				LocationID:     ptr(sl.LocationID),
				Name:           sl.Name,
				Latitude:       ptr(sl.Latitude),
				Longitude:      ptr(sl.Longitude),
				ShoreNormalDeg: sl.ShoreNormalDeg,
			})
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("%w: provide at least one spot or save a location first", domain.ErrValidation)
		}
		return out, nil
	}

	out := make([]domain.Spot, 0, len(spots))
	for i, spot := range spots {
		resolved, err := s.resolveSpot(ctx, spot)
		if err != nil {
			return nil, fmt.Errorf("spot %d: %w", i, err)
		}
		out = append(out, resolved)
	}
	return out, nil
}

func (s *WindsurfService) resolveSpot(ctx context.Context, spot domain.Spot) (domain.Spot, error) {
	if spot.LocationID != nil && *spot.LocationID > 0 && spot.Latitude == nil {
		loc, err := s.catalog.ByID(ctx, *spot.LocationID)
		if err != nil {
			return domain.Spot{}, fmt.Errorf("resolve location %d: %w", *spot.LocationID, err)
		}
		spot.Name = loc.Name
		spot.Latitude = ptr(loc.Latitude)
		spot.Longitude = ptr(loc.Longitude)
	}
	if spot.Latitude == nil || spot.Longitude == nil {
		return domain.Spot{}, fmt.Errorf("%w: latitude and longitude are required", domain.ErrValidation)
	}
	if strings.TrimSpace(spot.Name) == "" {
		spot.Name = fmt.Sprintf("%.3f, %.3f", *spot.Latitude, *spot.Longitude)
	}
	return spot, nil
}

// windowResult couples the public recommendation with the internal
// "acceptable" flag used to decide the top-level message.
type windowResult struct {
	recommendation domain.Recommendation
	acceptable     bool
}

// bestWindow slides a fixed-length window over the hourly forecast and returns
// the strongest slot for the spot.
func bestWindow(spot domain.Spot, snapshot domain.WeatherSnapshot, profile domain.WindsurfProfile, slotHours int, now time.Time, staleThreshold time.Duration) (windowResult, bool) {
	hours := snapshot.Hourly
	if len(hours) < slotHours {
		return windowResult{}, false
	}

	isStale := snapshot.Stale(now, staleThreshold)
	best := windowResult{}
	for i := 0; i+slotHours <= len(hours); i++ {
		window := hours[i : i+slotHours]
		if window[0].Time.Before(now) {
			continue
		}
		rec := scoreWindow(spot, window, profile, isStale)
		if best.recommendation.Start.IsZero() || rec.recommendation.Score > best.recommendation.Score {
			best = rec
		}
	}
	if best.recommendation.Start.IsZero() {
		return windowResult{}, false
	}
	return best, true
}

func scoreWindow(spot domain.Spot, window []domain.HourlyForecast, profile domain.WindsurfProfile, isStale bool) windowResult {
	speeds := make([]float64, 0, len(window))
	dirs := make([]float64, 0, len(window))
	gusts := make([]float64, 0, len(window))
	for _, h := range window {
		speeds = append(speeds, h.WindSpeed)
		dirs = append(dirs, h.WindDirection)
		gusts = append(gusts, h.WindGusts)
	}

	avgSpeed := mean(speeds)
	avgDir := circularMean(dirs)
	avgGusts := mean(gusts)
	orientation := classifyOrientation(avgDir, spot.ShoreNormalDeg)

	ideal := (profile.WindSpeedMin + profile.WindSpeedMax) / 2
	windScore := rangeScore(avgSpeed, profile.WindSpeedMin, profile.WindSpeedMax, &ideal)
	orientScore := orientationScore(profile.Level, orientation)
	gustPenalty := gustPenalty(profile.Level, avgSpeed, avgGusts)

	score := round1(clamp(windScoreWeight*windScore+orientationScoreWeight*orientScore-gustPenalty, 0, scoreCeiling))
	acceptable := avgSpeed >= profile.WindSpeedMin && avgSpeed <= profile.WindSpeedMax

	rec := domain.Recommendation{
		SpotName:             spot.Name,
		LocationID:           spot.LocationID,
		Latitude:             *spot.Latitude,
		Longitude:            *spot.Longitude,
		Start:                window[0].Time,
		End:                  window[len(window)-1].Time.Add(time.Hour),
		Score:                score,
		AverageWindSpeed:     round1(avgSpeed),
		AverageWindDirection: round1(avgDir),
		AverageWindGusts:     round1(avgGusts),
		Orientation:          orientation,
		IsStale:              isStale,
		Reasons:              reasons(profile, avgSpeed, avgDir, avgGusts, orientation, acceptable),
	}
	return windowResult{recommendation: rec, acceptable: acceptable}
}

func reasons(profile domain.WindsurfProfile, speed, dir, gusts float64, o domain.Orientation, acceptable bool) []string {
	out := []string{
		fmt.Sprintf("Average wind %.1f m/s (target %.1f–%.1f m/s).",
			speed, profile.WindSpeedMin, profile.WindSpeedMax),
		fmt.Sprintf("Wind direction %.0f° classified as %s relative to the shore.", dir, o),
	}
	if gusts > speed*gustWarningRatio {
		out = append(out, fmt.Sprintf("Gusts reach %.1f m/s, which may be demanding for a %s rider.", gusts, profile.Level))
	}
	if acceptable {
		out = append(out, "Conditions fall within the requested range.")
	} else {
		out = append(out, "Conditions are outside the requested wind range but remain the closest option.")
	}
	return out
}

func classifyOrientation(windDir float64, shoreNormal *float64) domain.Orientation {
	if shoreNormal == nil {
		return domain.OrientationUnknown
	}
	d := angularDistance(windDir, *shoreNormal)
	switch {
	case d <= onshoreMaxDelta:
		return domain.OrientationOnshore
	case d >= offshoreMinDelta:
		return domain.OrientationOffshore
	default:
		return domain.OrientationCrossShore
	}
}

func orientationScore(level domain.SkillLevel, o domain.Orientation) float64 {
	switch o {
	case domain.OrientationOnshore:
		return orientationScoreOnshore
	case domain.OrientationCrossShore:
		if level == domain.LevelBeginner {
			return orientationScoreCrossBeginner
		}
		return orientationScoreCrossSkill
	case domain.OrientationOffshore:
		switch level {
		case domain.LevelBeginner:
			return orientationScoreOffshoreBegin
		case domain.LevelIntermediate:
			return orientationScoreOffshoreInt
		case domain.LevelAdvanced:
			return orientationScoreOffshoreAdv
		default:
			return orientationScoreOffshorePro
		}
	default:
		return orientationScoreUnknown
	}
}

func gustPenalty(level domain.SkillLevel, speed, gusts float64) float64 {
	if speed <= 0 || gusts <= speed {
		return 0
	}
	excess := (gusts - speed) / speed
	switch level {
	case domain.LevelBeginner:
		return clamp(excess*gustPenaltyScaleBeginner, 0, gustPenaltyCapBeginner)
	case domain.LevelIntermediate:
		return clamp(excess*gustPenaltyScaleIntermediate, 0, gustPenaltyCapIntermediate)
	default:
		return clamp(excess*gustPenaltyScaleAdvanced, 0, gustPenaltyCapAdvanced)
	}
}

func validateProfile(p *domain.WindsurfProfile) error {
	if !p.Level.Valid() {
		return fmt.Errorf("%w: level must be one of beginner, intermediate, advanced, pro", domain.ErrValidation)
	}
	if p.WindSpeedMin < 0 || p.WindSpeedMax <= 0 {
		return fmt.Errorf("%w: wind speed range must be positive", domain.ErrValidation)
	}
	if p.WindSpeedMin >= p.WindSpeedMax {
		return fmt.Errorf("%w: windSpeedMin must be lower than windSpeedMax", domain.ErrValidation)
	}
	return nil
}

func angularDistance(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func circularMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var x, y float64
	for _, v := range values {
		rad := v * math.Pi / 180
		x += math.Cos(rad)
		y += math.Sin(rad)
	}
	mean := math.Atan2(y, x) * 180 / math.Pi
	if mean < 0 {
		mean += 360
	}
	return mean
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func clampInt(v, fallback, lo, hi int) int {
	if v == 0 {
		return fallback
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func ptr[T any](v T) *T {
	return &v
}
