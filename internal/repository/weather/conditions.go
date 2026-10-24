package weather

// WMO weather interpretation codes used by Open-Meteo. Mapping them to human
// readable text keeps the weather condition payload independent from the
// upstream provider vocabulary.
//
// See https://open-meteo.com/en/docs (WMO Weather interpretation codes).
func conditionFromCode(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51, 53, 55:
		return "Drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61, 63, 65:
		return "Rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75, 77:
		return "Snow"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}

func iconFromCode(code int) string {
	switch {
	case code == 0:
		return "clear_sky"
	case code <= 3:
		return "cloudy_day"
	case code == 45 || code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 71 && code <= 77:
		return "snow"
	case code >= 80 && code <= 82:
		return "rain_showers"
	case code >= 85 && code <= 86:
		return "snow_showers"
	case code >= 95:
		return "thunderstorm"
	default:
		return "unknown"
	}
}
