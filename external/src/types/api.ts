export interface Location {
    id: number;
    name: string;
    latitude: number;
    longitude: number;
    countryCode: string;
    timezone?: string;
    createdAt?: string;
}

export interface SavedLocation {
    locationId: number;
    name: string;
    latitude: number;
    longitude: number;
    countryCode: string;
    timezone?: string;
    alias?: string;
    displayOrder: number;
    shoreNormalDeg?: number;
    savedAt: string;
}

export interface CurrentWeather {
    temperature: number;
    feelsLike: number;
    humidity: number;
    windSpeed: number;
    windDirection: number;
    windGusts: number;
    weatherCode: number;
    condition: string;
    icon: string;
    observedAt: string;
}

export interface MetricPreference {
    enabled: boolean;
    weight: number;
    min: number;
    max: number;
    ideal?: number;
}

export interface PrecipitationCriterion {
    enabled: boolean;
    weight: number;
    maxProbability: number;
}

export interface RankingCriteria {
    temperature: MetricPreference;
    wind: MetricPreference;
    precipitation: PrecipitationCriterion;
}

export interface MetricScore {
    metric: string;
    value: number;
    score: number;
    reason: string;
}

export interface RankedCity {
    rank: number;
    isLeader: boolean;
    locationId: number;
    name: string;
    countryCode: string;
    latitude: number;
    longitude: number;
    score: number;
    isStale: boolean;
    lastUpdated: string;
    weather: CurrentWeather;
    metrics: MetricScore[];
    reasons: string[];
}

export interface RankingResult {
    generatedAt: string;
    criteria: RankingCriteria;
    cities: RankedCity[];
}

export type SkillLevel = 'beginner' | 'intermediate' | 'advanced' | 'pro';

export interface WindsurfProfile {
    level: SkillLevel;
    windSpeedMin: number;
    windSpeedMax: number;
    updatedAt?: string;
}

export interface Spot {
    locationId?: number;
    name?: string;
    latitude?: number;
    longitude?: number;
    shoreNormalDeg?: number;
}

export interface Recommendation {
    isBest: boolean;
    spotName: string;
    locationId?: number;
    latitude: number;
    longitude: number;
    start: string;
    end: string;
    score: number;
    averageWindSpeed: number;
    averageWindDirection: number;
    averageWindGusts: number;
    orientation: 'onshore' | 'offshore' | 'cross-shore' | 'unknown';
    isStale: boolean;
    reasons: string[];
}

export interface RecommendationResult {
    generatedAt: string;
    profile: WindsurfProfile;
    recommendations: Recommendation[];
    message?: string;
}

export interface ActivityConditions {
    tempMin: number;
    tempMax: number;
    maxPrecipitationProbability: number;
    maxWindSpeed: number;
}

export interface ActivityTimeWindow {
    isBest: boolean;
    matchesConditions: boolean;
    start: string;
    end: string;
    score: number;
    averageTemperature: number;
    averageFeelsLike: number;
    maxPrecipitationProbability: number;
    averageWindSpeed: number;
    weatherCode: number;
    condition: string;
    reasons: string[];
}

export interface ActivityRecommendationResult {
    generatedAt: string;
    location: Location;
    date: string;
    activity: string;
    conditions: ActivityConditions;
    windows: ActivityTimeWindow[];
    message?: string;
}

