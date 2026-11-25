package handler_test

import (
	"net/http"
	"testing"
	"time"

	"weathertracker/internal/domain"
)

func TestIntegration_EndToEnd_UserStory1(t *testing.T) {
	api := newTestAPI(t)

	// Step 1: User Registration
	registerBody := map[string]string{
		"email":    "traveler@example.com",
		"password": "strong-password-123",
	}
	regRec := api.do(t, http.MethodPost, "/api/v1/auth/register", "", registerBody)
	if regRec.Code != http.StatusCreated {
		t.Fatalf("Step 1 failed: registration status = %d, want 201: %s", regRec.Code, regRec.Body.String())
	}
	regResp := decode[struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}](t, regRec)

	token := regResp.Token
	if token == "" {
		t.Fatal("Step 1 failed: empty auth token returned")
	}

	// Step 2: Location search via Geocoder
	searchRec := api.do(t, http.MethodGet, "/api/v1/locations/search?q=Berlin", "", nil)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("Step 2 failed: location search status = %d, want 200", searchRec.Code)
	}
	searchResp := decode[struct {
		Items []domain.Location `json:"items"`
	}](t, searchRec)
	if len(searchResp.Items) == 0 {
		t.Fatal("Step 2 failed: expected at least one location in search results")
	}
	berlin := searchResp.Items[0]

	// Step 3: Add locations to user tracked list
	addBerlinRec := api.do(t, http.MethodPost, "/api/v1/users/me/locations", token, map[string]any{
		"name":        berlin.Name,
		"latitude":    berlin.Latitude,
		"longitude":   berlin.Longitude,
		"countryCode": berlin.CountryCode,
	})
	if addBerlinRec.Code != http.StatusCreated {
		t.Fatalf("Step 3 failed: add Berlin status = %d, want 201: %s", addBerlinRec.Code, addBerlinRec.Body.String())
	}
	savedBerlin := decode[domain.SavedLocation](t, addBerlinRec)

	addMadridRec := api.do(t, http.MethodPost, "/api/v1/users/me/locations", token, map[string]any{
		"name":        "Madrid",
		"latitude":    40.41,
		"longitude":   -3.70,
		"countryCode": "ES",
	})
	if addMadridRec.Code != http.StatusCreated {
		t.Fatalf("Step 3 failed: add Madrid status = %d, want 201", addMadridRec.Code)
	}
	savedMadrid := decode[domain.SavedLocation](t, addMadridRec)

	// Step 4: Verify user saved locations list
	listRec := api.do(t, http.MethodGet, "/api/v1/users/me/locations", token, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("Step 4 failed: list saved locations status = %d, want 200", listRec.Code)
	}
	listResp := decode[struct {
		Limit int                    `json:"limit"`
		Items []domain.SavedLocation `json:"items"`
	}](t, listRec)
	if len(listResp.Items) != 2 {
		t.Fatalf("Step 4 failed: expected 2 saved locations, got %d", len(listResp.Items))
	}

	// Step 5: Execute User Story 1 - Rank cities by weather priorities
	rankBody := map[string]any{
		"locationIds": []int64{savedBerlin.LocationID, savedMadrid.LocationID},
		"criteria": map[string]any{
			"temperature": map[string]any{
				"enabled": true,
				"weight":  1.0,
				"min":     18.0,
				"max":     26.0,
				"ideal":   22.0,
			},
			"precipitation": map[string]any{
				"enabled":        true,
				"weight":         1.0,
				"maxProbability": 20.0,
			},
			"wind": map[string]any{
				"enabled": true,
				"weight":  0.5,
				"min":     1.0,
				"max":     6.0,
			},
		},
	}
	rankRec := api.do(t, http.MethodPost, "/api/v1/weather/rank", token, rankBody)
	if rankRec.Code != http.StatusOK {
		t.Fatalf("Step 5 failed: rank status = %d, want 200: %s", rankRec.Code, rankRec.Body.String())
	}
	rankResult := decode[domain.RankingResult](t, rankRec)

	if len(rankResult.Cities) != 2 {
		t.Fatalf("Step 5 failed: expected 2 ranked cities, got %d", len(rankResult.Cities))
	}
	if !rankResult.Cities[0].IsLeader {
		t.Fatalf("Step 5 failed: first city should be flagged as leader, got %+v", rankResult.Cities[0])
	}
	if rankResult.Cities[0].Score < rankResult.Cities[1].Score {
		t.Fatalf("Step 5 failed: cities must be in descending score order: %f < %f",
			rankResult.Cities[0].Score, rankResult.Cities[1].Score)
	}
	if len(rankResult.Cities[0].Reasons) == 0 {
		t.Fatal("Step 5 failed: expected explanatory reasons for ranking")
	}

	// Step 6: Remove location and verify deletion
	delRec := api.do(t, http.MethodDelete, "/api/v1/users/me/locations/"+itoa(savedBerlin.LocationID), token, nil)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("Step 6 failed: delete status = %d, want 204", delRec.Code)
	}

	afterDelRec := api.do(t, http.MethodGet, "/api/v1/users/me/locations", token, nil)
	afterDel := decode[struct {
		Items []domain.SavedLocation `json:"items"`
	}](t, afterDelRec)
	if len(afterDel.Items) != 1 || afterDel.Items[0].LocationID != savedMadrid.LocationID {
		t.Fatalf("Step 6 failed: expected only Madrid remaining, got %+v", afterDel.Items)
	}
}

func TestIntegration_EndToEnd_UserStory3(t *testing.T) {
	api := newTestAPI(t)

	// Step 1: User Registration & Token
	regRec := api.do(t, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":    "runner@example.com",
		"password": "running-pass-456",
	})
	if regRec.Code != http.StatusCreated {
		t.Fatalf("registration status = %d, want 201", regRec.Code)
	}
	token := decode[struct {
		Token string `json:"token"`
	}](t, regRec).Token

	// Step 2: Request outdoor activity recommendations for a chosen city and date
	today := time.Now().UTC().Format("2006-01-02")
	activityBody := map[string]any{
		"city": map[string]any{
			"name":        "Berlin",
			"latitude":    52.52,
			"longitude":   13.41,
			"countryCode": "DE",
		},
		"date":          today,
		"activity":      "running",
		"durationHours": 1,
		"conditions": map[string]any{
			"tempMin":                     5.0,
			"tempMax":                     25.0,
			"maxPrecipitationProbability": 30.0,
			"maxWindSpeed":                10.0,
		},
	}

	recRec := api.do(t, http.MethodPost, "/api/v1/activity/recommendations", token, activityBody)
	if recRec.Code != http.StatusOK {
		t.Fatalf("activity recommendation status = %d, want 200: %s", recRec.Code, recRec.Body.String())
	}

	result := decode[domain.ActivityRecommendationResult](t, recRec)
	if len(result.Windows) == 0 {
		t.Fatal("expected recommended time windows")
	}
	best := result.Windows[0]
	if !best.IsBest {
		t.Errorf("expected first window to be flagged as isBest")
	}
	if len(best.Reasons) < 3 {
		t.Errorf("expected detailed reasons (temperature, precipitation, wind), got %d", len(best.Reasons))
	}
}
