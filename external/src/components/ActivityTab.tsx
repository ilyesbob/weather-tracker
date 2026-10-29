import React, { useState } from 'react';
import { Clock, Activity, CheckCircle2, AlertTriangle, Loader2 } from 'lucide-react';
import { request } from '../api/client';
import { ActivityRecommendationResult } from '../types/api';
import {
    translateCondition,
    translateReason,
    translateBackendMessage,
    formatTimeOnlyRange,
} from '../utils/weather';

export const ActivityTab: React.FC = () => {
    const [activity, setActivity] = useState('running');
    const [date, setDate] = useState(() => new Date().toISOString().split('T')[0]);

    // Длительность: отдельно часы и минуты
    const [durationHours, setDurationHours] = useState(1);
    const [durationMinutes, setDurationMinutes] = useState(0);

    const [result, setResult] = useState<ActivityRecommendationResult | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const totalSelectedMinutes = durationHours * 60 + durationMinutes;

    const handleRecommend = async () => {
        setError('');

        if (totalSelectedMinutes <= 0) {
            setError('Укажите длительность больше 0 минут');
            return;
        }

        setLoading(true);
        try {
            // Бэкенд рассчитывает погоду по почасовым слотам (округляем вверх с запасом)
            const apiDurationHours = Math.max(1, Math.min(4, Math.ceil(totalSelectedMinutes / 60)));

            const data = await request<ActivityRecommendationResult>(
                '/api/v1/activity/recommendations',
                {
                    method: 'POST',
                    body: JSON.stringify({
                        activity,
                        date,
                        durationHours: apiDurationHours,
                    }),
                }
            );
            setResult(data);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="space-y-6">
            <div className="bg-slate-800 border border-slate-700 rounded-2xl p-6">
                <h2 className="text-xl font-bold mb-4 text-white flex items-center gap-2">
                    <Activity className="w-5 h-5 text-emerald-400" /> Подбор времени для тренировки / прогулки
                </h2>

                {error && (
                    <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg text-sm mb-4">
                        {error}
                    </div>
                )}

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Тип активности</label>
                        <select
                            value={activity}
                            onChange={(e) => setActivity(e.target.value)}
                            className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-emerald-500"
                        >
                            <option value="running">Бег</option>
                            <option value="cycling">Велосипед</option>
                            <option value="walking">Прогулка / Ходьба</option>
                        </select>
                    </div>

                    <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">Дата</label>
                        <input
                            type="date"
                            value={date}
                            min={new Date().toISOString().split('T')[0]}
                            onChange={(e) => setDate(e.target.value)}
                            className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white focus:outline-none focus:border-emerald-500"
                        />
                    </div>

                    {/* Выбор длительности: часы и минуты */}
                    <div>
                        <label className="block text-xs font-medium text-slate-400 mb-1">
                            Длительность тренировки
                        </label>
                        <div className="grid grid-cols-2 gap-2">
                            <div className="relative">
                                <input
                                    type="number"
                                    min={0}
                                    max={4}
                                    value={durationHours}
                                    onChange={(e) =>
                                        setDurationHours(Math.max(0, parseInt(e.target.value, 10) || 0))
                                    }
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white pr-7 focus:outline-none focus:border-emerald-500"
                                    placeholder="0"
                                />
                                <span className="absolute right-2.5 top-2 text-xs text-slate-400 pointer-events-none">
                  ч
                </span>
                            </div>
                            <div className="relative">
                                <input
                                    type="number"
                                    min={0}
                                    max={59}
                                    step={5}
                                    value={durationMinutes}
                                    onChange={(e) =>
                                        setDurationMinutes(
                                            Math.max(0, Math.min(59, parseInt(e.target.value, 10) || 0))
                                        )
                                    }
                                    className="w-full bg-slate-900 border border-slate-700 rounded-xl px-3 py-2 text-sm text-white pr-9 focus:outline-none focus:border-emerald-500"
                                    placeholder="0"
                                />
                                <span className="absolute right-2.5 top-2 text-xs text-slate-400 pointer-events-none">
                  мин
                </span>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="mt-5 flex justify-end">
                    <button
                        onClick={handleRecommend}
                        disabled={loading}
                        className="bg-emerald-600 hover:bg-emerald-500 text-white font-medium px-6 py-2.5 rounded-xl transition disabled:opacity-50 text-sm flex items-center gap-2"
                    >
                        {loading && <Loader2 className="w-4 h-4 animate-spin" />}
                        {loading ? 'Подбор слотов...' : 'Подобрать лучшее время'}
                    </button>
                </div>
            </div>

            {result && (
                <div className="space-y-4">
                    {result.message && (
                        <div className="p-4 bg-amber-500/10 border border-amber-500/20 text-amber-300 rounded-xl flex items-center gap-2 text-sm">
                            <AlertTriangle className="w-4 h-4 shrink-0" />
                            <span>{translateBackendMessage(result.message)}</span>
                        </div>
                    )}

                    <div className="grid grid-cols-1 gap-4">
                        {result.windows.map((win, i) => {
                            // Рассчитываем точное время окончания с учетом введенных минут
                            const preciseEndIso =
                                totalSelectedMinutes > 0
                                    ? new Date(
                                        new Date(win.start).getTime() + totalSelectedMinutes * 60 * 1000
                                    ).toISOString()
                                    : win.end;

                            return (
                                <div
                                    key={i}
                                    className={`p-5 rounded-2xl border transition ${
                                        win.isBest
                                            ? 'bg-slate-800 border-emerald-500 shadow-lg shadow-emerald-500/10'
                                            : 'bg-slate-800/60 border-slate-700'
                                    }`}
                                >
                                    <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
                                        <div>
                                            <h4 className="text-lg font-bold text-white flex items-center gap-2">
                                                <Clock className="w-4 h-4 text-emerald-400" />
                                                {/* Выводится точный интервал, например: 12:00 — 13:30 */}
                                                {formatTimeOnlyRange(win.start, preciseEndIso)}
                                                {win.isBest && (
                                                    <span className="bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 text-xs px-2.5 py-0.5 rounded-full font-medium">
                            Рекомендуемое время
                          </span>
                                                )}
                                            </h4>
                                            <p className="text-xs text-slate-400 mt-1">
                                                Погода: {translateCondition(win.condition, win.weatherCode)} (ощущается как{' '}
                                                {win.averageFeelsLike > 0 ? `+${win.averageFeelsLike}` : win.averageFeelsLike}
                                                °C)
                                            </p>
                                        </div>
                                        <div className="text-right">
                                            <div className="text-2xl font-black text-emerald-400">{win.score} / 100</div>
                                            <div className="text-xs text-slate-400">Балл комфорта</div>
                                        </div>
                                    </div>

                                    <div className="grid grid-cols-3 gap-3 bg-slate-900/80 p-3 rounded-xl my-3 text-center">
                                        <div>
                                            <div className="text-xs text-slate-400">Температура</div>
                                            <div className="text-base font-semibold text-white">
                                                {win.averageTemperature > 0
                                                    ? `+${win.averageTemperature}`
                                                    : win.averageTemperature}
                                                °C
                                            </div>
                                        </div>
                                        <div>
                                            <div className="text-xs text-slate-400">Осадки</div>
                                            <div className="text-base font-semibold text-white">
                                                {win.maxPrecipitationProbability}%
                                            </div>
                                        </div>
                                        <div>
                                            <div className="text-xs text-slate-400">Ветер</div>
                                            <div className="text-base font-semibold text-white">
                                                {win.averageWindSpeed} м/с
                                            </div>
                                        </div>
                                    </div>

                                    <div className="space-y-1.5 pt-1">
                                        {win.reasons.map((r, idx) => (
                                            <div key={idx} className="flex items-start gap-2 text-xs text-slate-300">
                                                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                                                <span>{translateReason(r)}</span>
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