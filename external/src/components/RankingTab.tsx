import React, { useState } from 'react';
import { Award, AlertTriangle, Thermometer, Wind, CloudRain, CheckCircle2, Loader2 } from 'lucide-react';
import { request } from '../api/client';
import { RankingCriteria, RankingResult } from '../types/api';
import {
    formatWindDirection,
    translateCondition,
    translateReason,
    formatDateTimeShort,
} from '../utils/weather';

export const RankingTab: React.FC = () => {
    const [criteria, setCriteria] = useState<RankingCriteria>({
        temperature: { enabled: true, weight: 1, min: 18, max: 26, ideal: 22 },
        wind: { enabled: true, weight: 1, min: 2, max: 8 },
        precipitation: { enabled: true, weight: 1, maxProbability: 20 },
    });

    const [result, setResult] = useState<RankingResult | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const runRanking = async () => {
        setLoading(true);
        setError('');
        try {
            const savedRes = await request<{ items: { locationId: number }[] }>(
                '/api/v1/users/me/locations'
            );
            const ids = (savedRes.items || []).map((x) => x.locationId);

            if (ids.length === 0) {
                throw new Error('Сначала добавьте города во вкладке "Мои места"');
            }

            const res = await request<RankingResult>('/api/v1/weather/rank', {
                method: 'POST',
                body: JSON.stringify({
                    locationIds: ids,
                    criteria,
                }),
            });
            setResult(res);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="space-y-6">
            {/* Форма критериев */}
            <div className="bg-slate-800 border border-slate-700 rounded-2xl p-6">
                <h2 className="text-xl font-bold mb-4 text-white">Критерии и приоритеты поездки</h2>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {/* Температура */}
                    <div className="p-4 bg-slate-900/60 border border-slate-700/60 rounded-xl space-y-3">
                        <div className="flex items-center justify-between">
              <span className="font-semibold text-white flex items-center gap-1.5 text-sm">
                <Thermometer className="w-4 h-4 text-orange-400" /> Температура (°C)
              </span>
                            <input
                                type="checkbox"
                                checked={criteria.temperature.enabled}
                                onChange={(e) =>
                                    setCriteria({
                                        ...criteria,
                                        temperature: { ...criteria.temperature, enabled: e.target.checked },
                                    })
                                }
                                className="rounded accent-sky-500"
                            />
                        </div>
                        <div className="grid grid-cols-2 gap-2 text-xs">
                            <div>
                                <label className="text-slate-400">Мин (°C)</label>
                                <input
                                    type="number"
                                    value={criteria.temperature.min}
                                    onChange={(e) =>
                                        setCriteria({
                                            ...criteria,
                                            temperature: { ...criteria.temperature, min: parseFloat(e.target.value) || 0 },
                                        })
                                    }
                                    className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white"
                                />
                            </div>
                            <div>
                                <label className="text-slate-400">Макс (°C)</label>
                                <input
                                    type="number"
                                    value={criteria.temperature.max}
                                    onChange={(e) =>
                                        setCriteria({
                                            ...criteria,
                                            temperature: { ...criteria.temperature, max: parseFloat(e.target.value) || 0 },
                                        })
                                    }
                                    className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white"
                                />
                            </div>
                        </div>
                    </div>

                    {/* Ветер */}
                    <div className="p-4 bg-slate-900/60 border border-slate-700/60 rounded-xl space-y-3">
                        <div className="flex items-center justify-between">
              <span className="font-semibold text-white flex items-center gap-1.5 text-sm">
                <Wind className="w-4 h-4 text-cyan-400" /> Ветер (м/с)
              </span>
                            <input
                                type="checkbox"
                                checked={criteria.wind.enabled}
                                onChange={(e) =>
                                    setCriteria({
                                        ...criteria,
                                        wind: { ...criteria.wind, enabled: e.target.checked },
                                    })
                                }
                                className="rounded accent-sky-500"
                            />
                        </div>
                        <div className="grid grid-cols-2 gap-2 text-xs">
                            <div>
                                <label className="text-slate-400">Мин (м/с)</label>
                                <input
                                    type="number"
                                    value={criteria.wind.min}
                                    onChange={(e) =>
                                        setCriteria({
                                            ...criteria,
                                            wind: { ...criteria.wind, min: parseFloat(e.target.value) || 0 },
                                        })
                                    }
                                    className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white"
                                />
                            </div>
                            <div>
                                <label className="text-slate-400">Макс (м/с)</label>
                                <input
                                    type="number"
                                    value={criteria.wind.max}
                                    onChange={(e) =>
                                        setCriteria({
                                            ...criteria,
                                            wind: { ...criteria.wind, max: parseFloat(e.target.value) || 0 },
                                        })
                                    }
                                    className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white"
                                />
                            </div>
                        </div>
                    </div>

                    {/* Осадки */}
                    <div className="p-4 bg-slate-900/60 border border-slate-700/60 rounded-xl space-y-3">
                        <div className="flex items-center justify-between">
              <span className="font-semibold text-white flex items-center gap-1.5 text-sm">
                <CloudRain className="w-4 h-4 text-blue-400" /> Вероятность осадков
              </span>
                            <input
                                type="checkbox"
                                checked={criteria.precipitation.enabled}
                                onChange={(e) =>
                                    setCriteria({
                                        ...criteria,
                                        precipitation: { ...criteria.precipitation, enabled: e.target.checked },
                                    })
                                }
                                className="rounded accent-sky-500"
                            />
                        </div>
                        <div>
                            <label className="text-slate-400 text-xs">Макс. вероятность (%)</label>
                            <input
                                type="number"
                                value={criteria.precipitation.maxProbability}
                                onChange={(e) =>
                                    setCriteria({
                                        ...criteria,
                                        precipitation: {
                                            ...criteria.precipitation,
                                            maxProbability: parseFloat(e.target.value) || 0,
                                        },
                                    })
                                }
                                className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white text-xs mt-1"
                            />
                        </div>
                    </div>
                </div>

                <div className="mt-5 flex justify-end">
                    <button
                        onClick={runRanking}
                        disabled={loading}
                        className="bg-sky-600 hover:bg-sky-500 text-white px-6 py-2.5 rounded-xl font-medium transition disabled:opacity-50 flex items-center gap-2 text-sm"
                    >
                        {loading && <Loader2 className="w-4 h-4 animate-spin" />}
                        {loading ? 'Расчёт рейтинга...' : 'Рассчитать рейтинг'}
                    </button>
                </div>
            </div>

            {error && (
                <div className="p-4 bg-red-500/10 border border-red-500/20 text-red-400 rounded-xl text-sm">
                    {error}
                </div>
            )}

            {/* Результаты сравнения */}
            {result && (
                <div className="space-y-4">
                    <h3 className="text-xl font-bold text-white">Результаты сравнения городов</h3>
                    <div className="grid grid-cols-1 gap-4">
                        {result.cities.map((city) => {
                            // Собираем все причины (из reasons или из metrics)
                            const reasonsList =
                                city.reasons && city.reasons.length > 0
                                    ? city.reasons
                                    : (city.metrics || []).map((m) => m.reason).filter(Boolean);

                            return (
                                <div
                                    key={city.locationId}
                                    className={`p-5 rounded-2xl border transition-all ${
                                        city.isLeader
                                            ? 'bg-slate-800 border-emerald-500 shadow-lg shadow-emerald-500/10'
                                            : 'bg-slate-800/60 border-slate-700'
                                    }`}
                                >
                                    <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
                                        <div className="flex items-center gap-3">
                      <span
                          className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm ${
                              city.isLeader ? 'bg-emerald-500 text-slate-950' : 'bg-slate-700 text-white'
                          }`}
                      >
                        #{city.rank}
                      </span>
                                            <div>
                                                <h4 className="text-lg font-bold text-white flex items-center gap-2">
                                                    {city.name}
                                                    {city.isLeader && (
                                                        <span className="flex items-center gap-1 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs px-2.5 py-0.5 rounded-full font-medium">
                              <Award className="w-3.5 h-3.5" /> Лидер рейтинга
                            </span>
                                                    )}
                                                    {city.isStale && (
                                                        <span className="flex items-center gap-1 bg-amber-500/10 border border-amber-500/30 text-amber-400 text-xs px-2 py-0.5 rounded-full font-medium">
                              <AlertTriangle className="w-3.5 h-3.5" /> Устаревшие данные
                            </span>
                                                    )}
                                                </h4>
                                                <p className="text-xs text-slate-400 mt-0.5">
                                                    Обновлено: {formatDateTimeShort(city.lastUpdated)}
                                                </p>
                                            </div>
                                        </div>

                                        <div className="text-right">
                                            <div className="text-2xl font-black text-sky-400">{city.score} / 100</div>
                                            <div className="text-xs text-slate-400">Балл соответствия</div>
                                        </div>
                                    </div>

                                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-slate-900/80 p-3 rounded-xl my-3 text-center">
                                        <div>
                                            <div className="text-xs text-slate-400">Температура</div>
                                            <div className="text-base font-semibold text-white">
                                                {city.weather.temperature > 0 ? `+${city.weather.temperature}` : city.weather.temperature}°C
                                            </div>
                                        </div>
                                        <div>
                                            <div className="text-xs text-slate-400">Ощущается как</div>
                                            <div className="text-base font-semibold text-white">
                                                {city.weather.feelsLike > 0 ? `+${city.weather.feelsLike}` : city.weather.feelsLike}°C
                                            </div>
                                        </div>
                                        <div>
                                            <div className="text-xs text-slate-400">Ветер</div>
                                            <div className="text-base font-semibold text-white">
                                                {city.weather.windSpeed} м/с
                                                <span className="block text-xs text-sky-400 font-normal">
                          {formatWindDirection(city.weather.windDirection)}
                        </span>
                                            </div>
                                        </div>
                                        <div>
                                            <div className="text-xs text-slate-400">Погода</div>
                                            <div className="text-base font-semibold text-white truncate">
                                                {translateCondition(city.weather.condition, city.weather.weatherCode)}
                                            </div>
                                        </div>
                                    </div>

                                    <div className="space-y-1.5 pt-1">
                                        {reasonsList.map((reason, idx) => (
                                            <div key={idx} className="flex items-start gap-2 text-xs text-slate-300">
                                                <CheckCircle2 className="w-3.5 h-3.5 text-sky-400 shrink-0 mt-0.5" />
                                                <span>{translateReason(reason)}</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}
        </div>
    );
};