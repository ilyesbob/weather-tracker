import React from 'react';
import { CloudSun, UserCheck, LogOut, LogIn } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

interface NavbarProps {
    activeTab: string;
    setActiveTab: (tab: string) => void;
    onOpenAuth: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ activeTab, setActiveTab, onOpenAuth }) => {
    const { user, isAuthenticated, logout } = useAuth();

    const tabs = [
        { id: 'locations', label: 'Мои места' },
        { id: 'ranking', label: 'Сравнение городов' },
        { id: 'windsurf', label: 'Виндсёрфинг' },
        { id: 'activity', label: 'Активности и тренировки' },
    ];

    return (
        <header className="bg-slate-800/90 backdrop-blur border-b border-slate-700 sticky top-0 z-40">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <CloudSun className="w-8 h-8 text-sky-400" />
                    <span className="font-bold text-lg tracking-wide text-white">WeatherTracker</span>
                </div>

                {isAuthenticated && (
                    <nav className="flex items-center gap-1 sm:gap-2">
                        {tabs.map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => setActiveTab(tab.id)}
                                className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                                    activeTab === tab.id
                                        ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                                        : 'text-slate-400 hover:text-slate-200 hover:bg-slate-700/50'
                                }`}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </nav>
                )}

                <div className="flex items-center gap-4">
                    {isAuthenticated ? (
                        <div className="flex items-center gap-3">
              <span className="text-xs sm:text-sm text-slate-300 flex items-center gap-1.5">
                <UserCheck className="w-4 h-4 text-emerald-400" />
                  {user?.email}
              </span>
                            <button
                                onClick={logout}
                                className="p-2 text-slate-400 hover:text-red-400 hover:bg-slate-700/50 rounded-lg transition"
                                title="Выйти"
                            >
                                <LogOut className="w-4 h-4" />
                            </button>
                        </div>
                    ) : (
                        <button
                            onClick={onOpenAuth}
                            className="flex items-center gap-2 bg-sky-600 hover:bg-sky-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition"
                        >
                            <LogIn className="w-4 h-4" />
                            Войти
                        </button>
                    )}
                </div>
            </div>
        </header>
    );
};

