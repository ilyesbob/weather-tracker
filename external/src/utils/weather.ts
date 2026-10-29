// 1. Функция форматирования направления ветра: "ЮВ (135°)"
export function formatWindDirection(deg: number | undefined | null): string {
    if (deg === undefined || deg === null || isNaN(deg)) return '';
    const normalized = ((deg % 360) + 360) % 360;
    const directions = ['С', 'СВ', 'В', 'ЮВ', 'Ю', 'ЮЗ', 'З', 'СЗ'];
    const idx = Math.round(normalized / 45) % 8;
    return `${directions[idx]} (${Math.round(normalized)}°)`;
}

// Алиас (на случай, если где-то вызвано во множественном числе)
export const formatWindDirections = formatWindDirection;

// 2. Форматирование интервала времени с датой: "30.10.2026, 15:00 — 18:00"
export function formatSlotRange(startIso: string, endIso: string): string {
    const s = new Date(startIso);
    const e = new Date(endIso);
    const datePart = s.toLocaleDateString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
    });
    const startTime = s.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    const endTime = e.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    return `${datePart}, ${startTime} — ${endTime}`;
}

// 3. Форматирование только времени: "15:00 — 16:00"
export function formatTimeOnlyRange(startIso: string, endIso: string): string {
    const s = new Date(startIso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    const e = new Date(endIso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
    return `${s} — ${e}`;
}

// 4. Форматирование времени обновления: "15:42"
export function formatDateTimeShort(iso: string): string {
    if (!iso) return '';
    const d = new Date(iso);
    return d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
}

// 5. Перевод ориентации ветра для виндсёрфинга
export function translateOrientation(orientation: string): string {
    switch (orientation?.toLowerCase()) {
        case 'onshore': return 'В берег (Onshore)';
        case 'offshore': return 'С берега (Offshore)';
        case 'cross-shore': return 'Вдоль берега (Cross-shore)';
        default: return orientation || 'Не определено';
    }
}

// 6. Перевод погодных условий
export function translateCondition(condition: string, code?: number): string {
    if (code !== undefined) {
        switch (code) {
            case 0: return 'Ясно';
            case 1: return 'Преимущественно ясно';
            case 2: return 'Переменная облачность';
            case 3: return 'Пасмурно';
            case 45: case 48: return 'Туман';
            case 51: case 53: case 55: return 'Морось';
            case 61: case 63: case 65: return 'Дождь';
            case 71: case 73: case 75: return 'Снегопад';
            case 77: return 'Снежные зёрна';
            case 80: case 81: case 82: return 'Ливневый дождь';
            case 85: case 86: return 'Снежный ливень';
            case 95: case 96: case 99: return 'Гроза';
        }
    }

    const dict: Record<string, string> = {
        'clear sky': 'Ясно',
        'clear': 'Ясно',
        'mainly clear': 'Преимущественно ясно',
        'partly cloudy': 'Переменная облачность',
        'overcast': 'Пасмурно',
        'cloudy': 'Облачно',
        'fog': 'Туман',
        'drizzle': 'Морось',
        'light rain': 'Небольшой дождь',
        'moderate rain': 'Умеренный дождь',
        'heavy rain': 'Сильный ливень',
        'rain': 'Дождь',
        'snow': 'Снег',
        'thunderstorm': 'Гроза',
    };

    return dict[condition?.toLowerCase().trim()] || condition || 'Без осадков';
}

// 7. Перевод служебных сообщений бэкенда
export function translateBackendMessage(msg: string): string {
    if (!msg) return '';
    if (/no exact match/i.test(msg) || /closest available/i.test(msg)) {
        return 'Точных совпадений по вашим критериям не найдено. Показаны наиболее близкие доступные интервалы.';
    }
    if (/no spots found/i.test(msg) || /no recommendations/i.test(msg)) {
        return 'Подходящих вариантов под заданные условия на эти даты не найдено.';
    }
    if (/data is stale/i.test(msg)) {
        return 'Данные о погоде устарели и обновляются.';
    }
    return translateReason(msg);
}

// 8. Перевод причин бэкенда
export function translateReason(text: string): string {
    if (!text) return '';

    const clean = text.trim();

    // Температура
    const tempComfort = clean.match(/Temperature\s+is\s+([\d.-]+)°?C?\s*\(comfort band\s+([\d.-]+)[–-]([\d.-]+)°?C?\)/i);
    if (tempComfort) {
        return `Температура ${tempComfort[1]}°C (комфортный диапазон ${tempComfort[2]}–${tempComfort[3]}°C).`;
    }

    const tempIdeal = clean.match(/Temperature\s+is\s+([\d.-]+)°?C?\s*\(ideal\s+([\d.-]+)°?C?\)/i);
    if (tempIdeal) {
        return `Температура ${tempIdeal[1]}°C (идеально ${tempIdeal[2]}°C).`;
    }

    // Осадки
    const precipMax = clean.match(/Precipitation probability\s+is\s+([\d.]+)%\s*\(max allowed\s+([\d.]+)%\)/i);
    if (precipMax) {
        return `Вероятность осадков ${precipMax[1]}% (максимально допустимо ${precipMax[2]}%).`;
    }

    // Ветер
    const windMax = clean.match(/Wind speed\s+is\s+([\d.]+)\s*m\/s\s*\(max allowed\s+([\d.]+)\s*m\/s\)/i);
    if (windMax) {
        return `Скорость ветра ${windMax[1]} м/с (максимально допустимо ${windMax[2]} м/с).`;
    }

    const windComfort = clean.match(/Wind speed\s+is\s+([\d.]+)\s*m\/s\s*\(comfort band\s+([\d.]+)[–-]([\d.]+)\s*m\/s\)/i);
    if (windComfort) {
        return `Скорость ветра ${windComfort[1]} м/с (комфортный диапазон ${windComfort[2]}–${windComfort[3]} м/с).`;
    }

    // Отклонения критериев
    if (/Conditions deviate from requested criteria.*?closest available time slot/i.test(clean)) {
        return 'Условия немного отклоняются от заданных критериев, но это наиболее подходящее время дня.';
    }

    if (/Conditions match all requested criteria/i.test(clean) || /Conditions fully match/i.test(clean)) {
        return 'Погодные условия полностью соответствуют всем заданным параметрам.';
    }

    // Виндсёрфинг
    const avgWindMatch = clean.match(/Average wind\s+([\d.]+)\s*m\/s\s*\(target\s*([\d.]+)[–-]([\d.]+)\s*m\/s\)/i);
    if (avgWindMatch) {
        return `Средний ветер ${avgWindMatch[1]} м/с (целевой диапазон ${avgWindMatch[2]}–${avgWindMatch[3]} м/с).`;
    }

    const windOrientMatch = clean.match(/Wind direction\s+(\d+)°\s+classified as\s+([\w-]+)\s+relative to the shore/i);
    if (windOrientMatch) {
        const deg = parseInt(windOrientMatch[1], 10);
        const dirFormatted = formatWindDirection(deg);
        const orient = translateOrientation(windOrientMatch[2]);
        return `Направление ветра ${dirFormatted}: классифицировано как «${orient.toLowerCase()}» относительно берега.`;
    }

    const gustsMatch = clean.match(/Gusts reach\s+([\d.]+)\s*m\/s.*?(beginner|intermediate|advanced|pro)?.*?rider/i);
    if (gustsMatch) {
        const levels: Record<string, string> = {
            beginner: 'начинающего',
            intermediate: 'среднего',
            advanced: 'опытного',
            pro: 'профессионального',
        };
        const lvl = levels[gustsMatch[2]?.toLowerCase()] || 'данного';
        return `Порывы достигают ${gustsMatch[1]} м/с, что может потребовать повышенного внимания для райдера ${lvl} уровня.`;
    }

    // Пословный запасной переводчик
    let fallback = clean
        .replace(/Temperature is\s*/gi, 'Температура: ')
        .replace(/Precipitation probability is\s*/gi, 'Вероятность осадков: ')
        .replace(/Wind speed is\s*/gi, 'Скорость ветра: ')
        .replace(/comfort band\s*/gi, 'комфортный диапазон: ')
        .replace(/max allowed\s*/gi, 'макс. допустимо: ')
        .replace(/ideal\s*/gi, 'идеально: ')
        .replace(/target\s*/gi, 'цель: ')
        .replace(/Conditions deviate from requested criteria/gi, 'Условия отклоняются от запрошенных критериев')
        .replace(/closest available time slot/gi, 'ближайший доступный интервал');

    return fallback;
}