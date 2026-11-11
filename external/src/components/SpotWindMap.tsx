import React from 'react';

interface SpotWindMapProps {
    latitude: number;
    longitude: number;
    spotName: string;
    windDirection: number; // Направление в градусах (0-360)
    windSpeed: number;     // Скорость в м/с
}

export const SpotWindMap: React.FC<SpotWindMapProps> = ({
                                                            latitude,
                                                            longitude,
                                                            spotName,
                                                            windDirection,
                                                            windSpeed,
                                                        }) => {
    // Вычисляем охват карты (bbox) с небольшим зумом вокруг спота
    const deltaLon = 0.04;
    const deltaLat = 0.025;
    const bbox = `${longitude - deltaLon}%2C${latitude - deltaLat}%2C${longitude + deltaLon}%2C${latitude + deltaLat}`;
    const osmEmbedUrl = `https://www.openstreetmap.org/export/embed.html?bbox=${bbox}&layer=mapnik`;

    // В метеорологии угол ветра — это ОТКУДА дует ветер.
    // Стрелка указывает, КУДА движется воздух (добавляем 180° к исходному углу)
    const flowRotation = (windDirection + 180) % 360;

    // Динамический цвет стрелок в зависимости от скорости ветра
    const getWindColor = (speed: number) => {
        if (speed < 4) return 'text-slate-400 stroke-slate-400';
        if (speed <= 8) return 'text-emerald-400 stroke-emerald-400';
        if (speed <= 15) return 'text-amber-400 stroke-amber-400';
        return 'text-rose-500 stroke-rose-500';
    };

    const windColorClass = getWindColor(windSpeed);

    // Сетка координат для расположения стрелок потока
    const arrowsGrid = [
        { top: '20%', left: '15%' },
        { top: '20%', left: '40%' },
        { top: '20%', left: '65%' },
        { top: '20%', left: '85%' },
        { top: '50%', left: '15%' },
        { top: '50%', left: '40%' },
        { top: '50%', left: '65%' },
        { top: '50%', left: '85%' },
        { top: '80%', left: '15%' },
        { top: '80%', left: '40%' },
        { top: '80%', left: '65%' },
        { top: '80%', left: '85%' },
    ];

    return (
        <div className="relative w-full h-56 sm:h-64 rounded-xl overflow-hidden border border-slate-700 bg-slate-950">
            {/*
              Кадрированный iframe:
              -top-12 (-48px) срезает кнопки зума сверху
              -left-10 (-40px) сдвигает влево
              w-[calc(100%+45px)] и h-[calc(100%+85px)] полностью выталкивают нижнюю плашку за нижний край
            */}
            <iframe
                title={`Карта ${spotName}`}
                src={osmEmbedUrl}
                loading="lazy"
                className="absolute -top-20 -left-10 w-[calc(100%+45px)] h-[calc(100%+140px)] max-w-none border-0 pointer-events-none filter contrast-105 brightness-100 opacity-90"
            />

            {/* Сетка стрелок направления ветра */}
            <div className="absolute inset-0 pointer-events-none">
                {arrowsGrid.map((pos, idx) => (
                    <div
                        key={idx}
                        className="absolute -translate-x-1/2 -translate-y-1/2 transition-transform duration-700 opacity-75"
                        style={{
                            top: pos.top,
                            left: pos.left,
                            transform: `translate(-50%, -50%) rotate(${flowRotation}deg)`,
                        }}
                    >
                        <svg
                            className={`w-5 h-5 ${windColorClass} drop-shadow-[0_1px_3px_rgba(0,0,0,0.9)]`}
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="black"
                            strokeWidth="2"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                        >
                            <line x1="12" y1="24" x2="12" y2="4" />
                            <polyline points="7 9 12 3 17 9" />
                        </svg>
                    </div>
                ))}
            </div>
        </div>
    );
};