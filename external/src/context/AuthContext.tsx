import React, { createContext, useContext, useState, useEffect } from 'react';
import { request } from '../api/client';

interface User {
    id: string;
    email: string;
    createdAt: string;
}

interface AuthResponse {
    token: string;
    user: User;
}

interface AuthContextType {
    token: string | null;
    user: User | null;
    login: (email: string, pass: string) => Promise<void>;
    register: (email: string, pass: string) => Promise<void>;
    logout: () => void;
    isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [token, setToken] = useState<string | null>(() => localStorage.getItem('wt_token'));
    const [user, setUser] = useState<User | null>(() => {
        const raw = localStorage.getItem('wt_user');
        return raw ? JSON.parse(raw) : null;
    });

    const saveAuth = (auth: AuthResponse) => {
        setToken(auth.token);
        setUser(auth.user);
        localStorage.setItem('wt_token', auth.token);
        localStorage.setItem('wt_user', JSON.stringify(auth.user));
    };

    const login = async (email: string, password: string) => {
        const data = await request<AuthResponse>('/api/v1/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        saveAuth(data);
    };

    const register = async (email: string, password: string) => {
        const data = await request<AuthResponse>('/api/v1/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        saveAuth(data);
    };

    const logout = () => {
        setToken(null);
        setUser(null);
        localStorage.removeItem('wt_token');
        localStorage.removeItem('wt_user');
    };

    return (
        <AuthContext.Provider value={{ token, user, login, register, logout, isAuthenticated: !!token }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => {
    const ctx = useContext(AuthContext);
    if (!ctx) throw new Error('useAuth must be used within AuthProvider');
    return ctx;
};

