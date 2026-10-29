const BASE_URL = import.meta.env.VITE_API_URL || '';

export async function request<T>(
    endpoint: string,
    options: RequestInit = {}
): Promise<T> {
    const token = localStorage.getItem('wt_token');

    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...(options.headers as Record<string, string>),
    };

    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${BASE_URL}${endpoint}`, {
        ...options,
        headers,
    });

    if (response.status === 204) {
        return {} as T;
    }

    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
        const errorMsg = data.error || `Ошибка сервера (${response.status})`;
        throw new Error(errorMsg);
    }

    return data as T;
}

