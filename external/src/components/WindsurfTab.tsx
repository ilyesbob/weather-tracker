import React, { useState, useEffect } from 'react';
import { Compass, Waves, Wind, CheckCircle2, AlertCircle } from 'lucide-react';
import { request } from '../api/client';
import { SkillLevel, WindsurfProfile, RecommendationResult } from '../types/api';
import {
    formatWindDirection,
    formatSlotRange,
    translateOrientation,
    translateReason,
} from '../utils/weather';

export const WindsurfTab: React.FC = () => {
    const [profile, setProfile] = useState<WindsurfProfile>({
        level: 'intermediate',
        windSpeedMin: 6,
        windSpeedMax: 14,
    });

    const [result, setResult] = useState<RecommendationResult | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [saveSuccess, setSaveSuccess] = useState(false);

    useEffect(() => {
        request<WindsurfProfile>('/api/v1/users/me/windsurf-profile')
            .then((data) => setProfile(data))
            .catch(() => {});
    }, []);

    const saveProfile = async () => {
        setError('');
        setSaveSuccess(false);
        try {
            await request('/api/v1/users/me/windsurf-profile', {
                method: 'PUT',
                body: JSON.stringify(profile),
            });
            setSaveSuccess(true);
            setTimeout(() => setSaveSuccess(false), 3000);
        } catch (err: any) {
            setError(err.message);
        }
    };

    const getRecommendations = async () => {
        setLoading(true);
        setError('');
        try {
            const data = await request<RecommendationResult>('/api/v1/windsurf/recommendations', {
                method: 'POST',
                body: JSON.stringify({
                    level: profile.level,
                    windSpeedMin: profile.windSpeedMin,
                    windSpeedMax: profile.windSpeedMax,
                    days: 3,
                    slotHours: 3,
                }),
            });
            setResult(data);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="space-y-6">
            {/* Профиль райдера и параметры ветра */}
            <div className="bg-slate-800 border border-slate-700 rounded-2xl p-6">
                <h2 className="text-xl font-bold mb-4 text-white flex items-center gap-2">
                    <Waves className="w-5 h-5 text-sky-400" /> Профиль райдера и параметры ветра
                </h2>

                {error && (
                    <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-400 rounded-lg text-sm mb-4">
                        {error}
                    </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {/* Уровень катания */}
                    <div className="p-4 bg-slate-900/60 border border-slate-700/60 rounded-xl space-y-3">
                        <div className="flex items-center justify-between">
              <span className="font-semibold text-white flex items-center gap-1.5 text-sm">
                <Waves className="w-4 h-4 text-sky-400" /> Уровень катания
              </span>
                        </div>
                        <div>
                            <label className="block text-xs text-slate-400 mb-1.5">Квалификация райдера</label>
                            <select
                                value={profile.level}
                                onChange={(e) => setProfile({ ...profile, level: e.target.value as SkillLevel })}
                                className="w-full bg-slate-800 border border-slate-700 rounded p-2 text-xs text-white focus:outline-none focus:border-sky-500"
                            >
                                <option value="beginner">Новичок (Beginner)</option>
                                <option value="intermediate">Средний (Intermediate)</option>
                                <option value="advanced">Продвинутый (Advanced)</option>
                                <option value="pro">Профи (Pro)</option>
                            </select>
                        </div>
                    </div>

                    {/* Ветер (м/с) — карточка в точности как на скриншоте */}
                    <div className="p-4 bg-slate-900/60 border border-slate-700/60 rounded-xl space-y-3">
                        <div className="flex items-center justify-between">
              <span className="font-semibold text-white flex items-center gap-1.5 text-sm">
                <Wind className="w-4 h-4 text-cyan-400" /> Ветер (м/с)
              </span>
                        </div>
                        <div className="grid grid-cols-2 gap-2 text-xs">
                            <div>
                                <label className="text-slate-400">Мин (м/с)</label>
                                <input
                                    type="number"
                                    value={profile.windSpeedMin}
                                    onChange={(e) =>
                                        setProfile({ ...profile, windSpeedMin: parseFloat(e.target.value) || 0 })
                                    }
                                    className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white"
                                />
                            </div>
                            <div>
                                <label className="text-slate-400">Макс (м/с)</label>
                                <input
                                    type="number"
                                    value={profile.windSpeedMax}
                                    onChange={(e) =>
                                        setProfile({ ...profile, windSpeedMax: parseFloat(e.target.value) || 0 })
                                    }
                                    className="w-full bg-slate-800 border border-slate-700 rounded p-1.5 text-white"
                                />
                            </div>
                        </div>
                    </div>
                </div>

                <div className="mt-4 flex flex-wrap items-center gap-3">
                    <button
                        onClick={saveProfile}
                        className="bg-slate-700 hover:bg-slate-600 text-white text-sm font-medium px-4 py-2 rounded-xl transition"
                    >
                        Сохранить профиль
                    </button>
                    <button
                        onClick={getRecommendations}
                        disabled={loading}
                        className="bg-sky-600 hover:bg-sky-500 text-white text-sm font-medium px-5 py-2 rounded-xl transition disabled:opacity-50"
                    >
                        {loading ? 'Анализ спотов...' : 'Найти лучшие слоты и споты'}
                    </button>
                    {saveSuccess && (
                        <span className="text-emerald-400 text-xs self-center">✓ Профиль сохранён</span>
                    )}
                </div>
            </div>

            {/* Результаты рекомендаций спотов */}
            {result && (
                <div className="space-y-4">
                    {result.message && (
                        <div className="p-4 bg-sky-500/10 border border-sky-500/20 text-sky-300 rounded-xl flex items-center gap-2 text-sm">
                            <AlertCircle className="w-4 h-4 shrink-0" />
                            <span>{result.message}</span>
                        </div>
                    )}

                    <div className="grid grid-cols-1 gap-4">
                        {result.recommendations.map((rec, i) => (
                            <div
                                key={i}
                                className={`p-5 rounded-2xl border transition ${
                                    rec.isBest
                                        ? 'bg-slate-800 border-sky-500 shadow-lg shadow-sky-500/10'
                                        : 'bg-slate-800/60 border-slate-700'
                                }`}
                            >
                                <div className="flex flex-wrap items-center justify-between gap-3 mb-3">
                                    <div>
                                        <h4 className="text-lg font-bold text-white flex items-center gap-2">
                                            {rec.spotName}
                                            {rec.isBest && (
                                                <span className="bg-sky-500/20 text-sky-300 border border-sky-500/30 text-xs px-2.5 py-0.5 rounded-full font-medium">
                          Лучший вариант
                        </span>
                                            )}
                                            <span
                                                className={`text-xs px-2.5 py-0.5 rounded-full font-medium border ${
                                                    rec.orientation === 'onshore'
                                                        ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                                                        : rec.orientation === 'offshore'
                                                            ? 'bg-rose-500/10 border-rose-500/30 text-rose-400'
                                                            : 'bg-amber-500/10 border-amber-500/30 text-amber-400'
                                                }`}
                                            >
                        {translateOrientation(rec.orientation)}
                      </span>
                                        </h4>
                                        <p className="text-xs text-slate-400 mt-1">
                                            Слот: {formatSlotRange(rec.start, rec.end)}
                                        </p>
                                    </div>
                                    <div className="text-right">
                                        <div className="text-2xl font-black text-sky-400">{rec.score} / 100</div>
                                        <div className="text-xs text-slate-400">Индекс условий</div>
                                    </div>
                                </div>

                                <div className="grid grid-cols-3 gap-3 bg-slate-900/80 p-3 rounded-xl my-3 text-center">
                                    <div>
                                        <div className="text-xs text-slate-400">Средняя скорость</div>
                                        <div className="text-base font-semibold text-white">
                                            {rec.averageWindSpeed} м/с
                                        </div>
                                    </div>
                                    <div>
                                        <div className="text-xs text-slate-400">Порывы</div>
                                        <div className="text-base font-semibold text-white">
                                            {rec.averageWindGusts} м/с
                                        </div>
                                    </div>
                                    <div>
                                        <div className="text-xs text-slate-400">Направление</div>
                                        <div className="text-base font-semibold text-white flex items-center justify-center gap-1">
                                            <Compass className="w-4 h-4 text-sky-400" />
                                            {formatWindDirection(rec.averageWindDirection)}
                                        </div>
                                    </div>
                                </div>

                                <div className="space-y-1.5 pt-1">
                                    {rec.reasons.map((reason, idx) => (
                                        <div key={idx} className="flex items-start gap-2 text-xs text-slate-300">
                                            <CheckCircle2 className="w-3.5 h-3.5 text-sky-400 shrink-0 mt-0.5" />
                                            <span>{translateReason(reason)}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
};