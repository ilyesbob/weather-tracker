import React, { useState, useEffect } from 'react';
import {
    Search,
    Plus,
    Trash2,
    MapPin,
    Compass,
    Thermometer,
    Wind,
    CloudRain,
    Navigation,
    Loader2,
} from 'lucide-react';
import { request } from '../api/client';
import { Location, SavedLocation, CurrentWeather, RankingResult } from '../types/api';
import { formatWindDirection, translateCondition } from '../utils/weather';

interface LocationWeatherSummary {
    weather: CurrentWeather;
    precipitationProbability: number;
}

export const LocationsTab: React.FC = () => {
    const [saved, setSaved] = useState<SavedLocation[]>([]);
    const [weatherMap, setWeatherMap] = useState<Record<number, LocationWeatherSummary>>({});
    const [weatherLoading, setWeatherLoading] = useState(false);

    // Режим добавления: 'search' (поиск) или 'custom' (своя точка)
    const [addMode, setAddMode] = useState<'search' | 'custom'>('search');

    // Поиск по названию
    const [query, setQuery] = useState('');
    const [searchResults, setSearchResults] = useState<Location[]>([]);
    const [searchLoading, setSearchLoading] = useState(false);

    // Ручной ввод точки
    const [customName, setCustomName] = useState('');
    const [customLat, setCustomLat] = useState('');
    const [customLon, setCustomLon] = useState('');
    const [customCountry, setCustomCountry] = useState('RU');
    const [customShoreNormal, setCustomShoreNormal] = useState('');

    const [error, setError] = useState('');

    const loadSavedLocations = async () => {
        try {
            const data = await request<{ items: SavedLocation[]; limit: number }>(
                '/api/v1/users/me/locations'
            );
            const items = data.items || [];
            setSaved(items);

            if (items.length > 0) {
                fetchWeatherForLocations(items.map((x) => x.locationId));
            } else {
                setWeatherMap({});
            }
        } catch (err: any) {
            setError(err.message);
        }
    };

    const fetchWeatherForLocations = async (locationIds: number[]) => {
        setWeatherLoading(true);
        try {
            const res = await request<RankingResult>('/api/v1/weather/rank', {
                method: 'POST',
                body: JSON.stringify({
                    locationIds,
                    criteria: {
                        temperature: { enabled: true, weight: 1, min: -50, max: 50 },
                        wind: { enabled: true, weight: 1, min: 0, max: 50 },
                        precipitation: { enabled: true, weight: 1, maxProbability: 100 },
                    },
                }),
            });

            const map: Record<number, LocationWeatherSummary> = {};
            res.cities.forEach((city) => {
                const precipMetric = city.metrics.find((m) => m.metric === 'precipitation');
                map[city.locationId] = {
                    weather: city.weather,
                    precipitationProbability: precipMetric ? Math.round(precipMetric.value) : 0,
                };
            });
            setWeatherMap(map);
        } catch {
            // Игнорируем ошибку получения погоды, чтобы не ломать список мест
        } finally {
            setWeatherLoading(false);
        }
    };

    useEffect(() => {
        loadSavedLocations();
    }, []);

    // Поиск города
    const handleSearch = async (e: React.FormEvent) => {
        e.preventDefault();
        if (query.trim().length < 2) return;
        setSearchLoading(true);
        setError('');
        try {
            const data = await request<{ items: Location[] }>(
                `/api/v1/locations/search?q=${encodeURIComponent(query)}&limit=5`
            );
            setSearchResults(data.items || []);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setSearchLoading(false);
        }
    };

    // Добавление из результатов поиска
    const handleAddFound = async (loc: Location) => {
        setError('');
        try {
            await request('/api/v1/users/me/locations', {
                method: 'POST',
                body: JSON.stringify({
                    locationId: loc.id,
                    name: loc.name,
                    latitude: loc.latitude,
                    longitude: loc.longitude,
                    countryCode: loc.countryCode,
                }),
            });
            setSearchResults([]);
            setQuery('');
            await loadSavedLocations();
        } catch (err: any) {
            setError(err.message);
        }
    };

    // Добавление произвольной точки по координатам
    const handleAddCustom = async (e: React.FormEvent) => {
        e.preventDefault();
        setError('');

        const lat = parseFloat(customLat);
        const lon = parseFloat(customLon);

        if (!customName.trim()) {
            setError('Укажите название точки');
            return;
        }
        if (isNaN(lat) || lat < -90 || lat > 90) {
            setError('Широта должна быть в диапазоне от -90 до 90');
            return;
        }
        if (isNaN(lon) || lon < -180 || lon > 180) {
            setError('Долгота должна быть в диапазоне от -180 до 180');
            return;
        }

        try {
            await request('/api/v1/users/me/locations', {
                method: 'POST',
                body: JSON.stringify({
                    name: customName.trim(),
                    latitude: lat,
                    longitude: lon,
                    countryCode: customCountry.trim().toUpperCase() || 'RU',
                    shoreNormalDeg: customShoreNormal ? parseFloat(customShoreNormal) : undefined,
                }),
            });

            setCustomName('');
            setCustomLat('');
            setCustomLon('');
            setCustomShoreNormal('');
            await loadSavedLocations();
        } catch (err: any) {
            setError(err.message);
        }
    };

    const deleteLocation = async (id: number) => {
        try {
            await request(`/api/v1/users/me/locations/${id}`, { method: 'DELETE' });
            await loadSavedLocations();
        } catch (err: any) {
            setError(err.message);
        }
    };

    return (
        <div className="space-y-6">
            {/* Карточки сохранённых мест */}
            <div className="bg-slate-800 border border-slate-700 rounded-2xl p-6">
                <div className="flex items-center justify-between mb-4">
                    <div>
                        <h2 className="text-xl font-bold text-white">Сохранённые места</h2>
                        <p className="text-xs text-slate-400 mt-0.5">
                            Текущая погода и условия для быстрого обзора
                        </p>
                    </div>
                    <span className="text-xs font-semibold px-3 py-1 bg-slate-700 rounded-full text-slate-300">
            {saved.length} / 5 мест
          </span>
                </div>

                {error && (
                    <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl text-sm mb-4">
                        {error}
                    </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {saved.map((loc) => {
                        const summary = weatherMap[loc.locationId];

                        return (
                            <div
                                key={loc.locationId}
                                className="bg-slate-900/80 border border-slate-700 rounded-xl p-4 flex flex-col justify-between hover:border-slate-600 transition shadow-sm"
                            >
                                <div>
                                    <div className="flex items-start justify-between gap-2">
                                        <div>
                                            <h4 className="font-bold text-white text-base flex items-center gap-1.5">
                                                <MapPin className="w-4 h-4 text-sky-400 shrink-0" />
                                                {loc.name}
                                                <span className="text-xs text-slate-400 font-normal">
                          ({loc.countryCode})
                        </span>
                                            </h4>
                                            <div className="text-xs text-slate-500 mt-0.5">
                                                {loc.latitude.toFixed(3)}°, {loc.longitude.toFixed(3)}°
                                            </div>
                                        </div>
                                        <button
                                            onClick={() => deleteLocation(loc.locationId)}
                                            className="text-slate-500 hover:text-red-400 p-1.5 rounded-lg hover:bg-slate-800 transition"
                                            title="Удалить точку"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>

                                    {loc.shoreNormalDeg !== undefined && (
                                        <div className="text-xs text-emerald-400 mt-1 flex items-center gap-1">
                                            <Compass className="w-3.5 h-3.5" />
                                            Береговая нормаль: {loc.shoreNormalDeg}°
                                        </div>
                                    )}

                                    {/* Сводка погоды */}
                                    <div className="mt-3 pt-3 border-t border-slate-800">
                                        {summary ? (
                                            <div className="space-y-2">
                                                <div className="flex items-center justify-between">
                          <span className="text-xs font-medium text-sky-300">
                            {translateCondition(
                                summary.weather.condition,
                                summary.weather.weatherCode
                            )}
                          </span>
                                                    <span className="text-lg font-bold text-white">
                            {summary.weather.temperature > 0
                                ? `+${summary.weather.temperature}`
                                : summary.weather.temperature}
                                                        °C
                          </span>
                                                </div>

                                                <div className="grid grid-cols-2 gap-2 text-xs text-slate-300 bg-slate-800/60 p-2 rounded-lg">
                                                    <div className="flex items-center gap-1.5">
                                                        <Wind className="w-3.5 h-3.5 text-cyan-400 shrink-0" />
                                                        <span>
                              {summary.weather.windSpeed} м/с,{' '}
                                                            <strong className="text-white font-semibold">
                                {formatWindDirection(summary.weather.windDirection)}
                              </strong>
                            </span>
                                                    </div>
                                                    <div className="flex items-center gap-1.5">
                                                        <CloudRain className="w-3.5 h-3.5 text-blue-400 shrink-0" />
                                                        <span>Осадки: {summary.precipitationProbability}%</span>
                                                    </div>
                                                </div>
                                            </div>
                                        ) : weatherLoading ? (
                                            <div className="flex items-center gap-2 text-xs text-slate-500 py-2">
                                                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                                                Загрузка погоды...
                                            </div>
                                        ) : (
                                            <div className="text-xs text-slate-500 py-1 italic">
                                                Данные о погоде недоступны
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </div>
                        );
                    })}

                    {saved.length === 0 && (
                        <div className="col-span-full py-8 text-center text-slate-400 text-sm">
                            Список пуст. Вы можете найти город по названию или указать произвольные координаты.
                        </div>
                    )}
                </div>
            </div>

            {/* Панель добавления места */}
            <div className="bg-slate-800 border border-slate-700 rounded-2xl p-6">
                {/* Переключатель режимов */}
                <div className="flex items-center gap-2 mb-4 p-1 bg-slate-900 rounded-xl w-fit border border-slate-700">
                    <button
                        type="button"
                        onClick={() => setAddMode('search')}
                        className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition ${
                            addMode === 'search'
                                ? 'bg-sky-600 text-white shadow'
                                : 'text-slate-400 hover:text-slate-200'
                        }`}
                    >
                        Поиск города по названию
                    </button>
                    <button
                        type="button"
                        onClick={() => setAddMode('custom')}
                        className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition ${
                            addMode === 'custom'
                                ? 'bg-sky-600 text-white shadow'
                                : 'text-slate-400 hover:text-slate-200'
                        }`}
                    >
                        Своя точка по координатам
                    </button>
                </div>

                {/* Режим 1: Поиск города */}
                {addMode === 'search' ? (
                    <div>
                        <form onSubmit={handleSearch} className="flex gap-2">
                            <div className="relative flex-1">
                                <Search className="w-4 h-4 absolute left-3.5 top-3.5 text-slate-400" />
                                <input
                                    type="text"
                                    value={query}
                                    onChange={(e) => setQuery(e.target.value)}
                                    placeholder="Введите название..."
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl pl-10 pr-4 py-2.5 text-sm text-white focus:outline-none focus:border-sky-500"
                                />
                            </div>
                            <button
                                type="submit"
                                disabled={searchLoading || query.trim().length < 2}
                                className="bg-sky-600 hover:bg-sky-500 text-white px-5 py-2.5 font-medium rounded-xl text-sm transition disabled:opacity-50 flex items-center gap-2"
                            >
                                {searchLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Найти'}
                            </button>
                        </form>

                        {searchResults.length > 0 && (
                            <div className="mt-4 divide-y divide-slate-700 border border-slate-700 rounded-xl overflow-hidden bg-slate-900">
                                {searchResults.map((res) => (
                                    <div
                                        key={res.id}
                                        className="p-3.5 flex items-center justify-between hover:bg-slate-800/50 transition"
                                    >
                                        <div>
                                            <div className="font-medium text-white text-sm">{res.name}</div>
                                            <div className="text-xs text-slate-400">
                                                {res.countryCode} • {res.latitude.toFixed(3)}°, {res.longitude.toFixed(3)}°
                                            </div>
                                        </div>
                                        <button
                                            onClick={() => handleAddFound(res)}
                                            disabled={saved.length >= 5}
                                            className="flex items-center gap-1.5 bg-emerald-600 hover:bg-emerald-500 text-white px-3 py-1.5 rounded-lg text-xs font-medium transition disabled:opacity-50"
                                        >
                                            <Plus className="w-3.5 h-3.5" /> Добавить
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                ) : (
                    /* Режим 2: Ручной ввод координат */
                    <form onSubmit={handleAddCustom} className="space-y-4">
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
                            <div className="lg:col-span-2">
                                <label className="block text-xs font-medium text-slate-400 mb-1">
                                    Название точки
                                </label>
                                <input
                                    type="text"
                                    required
                                    value={customName}
                                    onChange={(e) => setCustomName(e.target.value)}
                                    placeholder="Например: Пляж Омега"
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-sky-500"
                                />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-slate-400 mb-1">
                                    Широта (Latitude)
                                </label>
                                <input
                                    type="number"
                                    step="any"
                                    required
                                    value={customLat}
                                    onChange={(e) => setCustomLat(e.target.value)}
                                    placeholder="59.94"
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-sky-500"
                                />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-slate-400 mb-1">
                                    Долгота (Longitude)
                                </label>
                                <input
                                    type="number"
                                    step="any"
                                    required
                                    value={customLon}
                                    onChange={(e) => setCustomLon(e.target.value)}
                                    placeholder="30.31"
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-sky-500"
                                />
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-slate-400 mb-1">
                                    Береговая нормаль (°)
                                </label>
                                <input
                                    type="number"
                                    min="0"
                                    max="360"
                                    value={customShoreNormal}
                                    onChange={(e) => setCustomShoreNormal(e.target.value)}
                                    placeholder="0-360"
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-sky-500"
                                />
                            </div>
                        </div>

                        <div className="flex justify-end">
                            <button
                                type="submit"
                                disabled={saved.length >= 5}
                                className="bg-emerald-600 hover:bg-emerald-500 text-white px-5 py-2.5 font-medium rounded-xl text-sm transition disabled:opacity-50 flex items-center gap-2"
                            >
                                <Navigation className="w-4 h-4" /> Сохранить точку
                            </button>
                        </div>
                    </form>
                )}
            </div>
        </div>
    );
};

