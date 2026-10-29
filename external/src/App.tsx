import React, { useState } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { Navbar } from './components/Navbar';
import { AuthModal } from './components/AuthModal';
import { LocationsTab } from './components/LocationsTab';
import { RankingTab } from './components/RankingTab';
import { WindsurfTab } from './components/WindsurfTab';
import { ActivityTab } from './components/ActivityTab';

const Dashboard: React.FC = () => {
    const [activeTab, setActiveTab] = useState('locations');
    const [authModalOpen, setAuthModalOpen] = useState(false);
    const { isAuthenticated } = useAuth();

    return (
        <div className="min-h-screen flex flex-col bg-slate-900 text-slate-100">
            <Navbar
                activeTab={activeTab}
                setActiveTab={setActiveTab}
                onOpenAuth={() => setAuthModalOpen(true)}
            />

            <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 py-8">
                {!isAuthenticated ? (
                    <div className="text-center py-20 bg-slate-800/40 border border-slate-700/60 rounded-3xl p-8 max-w-2xl mx-auto">
                        <h2 className="text-3xl font-extrabold text-white mb-3">
                            Умный планировщик погоды
                        </h2>
                        <p className="text-slate-400 mb-6 leading-relaxed">
                            Сравнивайте города для поездок по гибким приоритетам (US1), ловите лучший ветер для виндсёрфинга (US2) или планируйте идеальные часы для тренировок на улице (US3).
                        </p>
                        <button
                            onClick={() => setAuthModalOpen(true)}
                            className="bg-sky-600 hover:bg-sky-500 text-white font-medium px-6 py-3 rounded-xl transition shadow-lg shadow-sky-600/20"
                        >
                            Войти или зарегистрироваться
                        </button>
                    </div>
                ) : (
                    <div>
                        {activeTab === 'locations' && <LocationsTab />}
                        {activeTab === 'ranking' && <RankingTab />}
                        {activeTab === 'windsurf' && <WindsurfTab />}
                        {activeTab === 'activity' && <ActivityTab />}
                    </div>
                )}
            </main>

            <AuthModal isOpen={authModalOpen} onClose={() => setAuthModalOpen(false)} />
        </div>
    );
};

export default function App() {
    return (
        <AuthProvider>
            <Dashboard />
        </AuthProvider>
    );
}

