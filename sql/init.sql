-- Создание таблицы инцидентов
CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    severity TEXT NOT NULL,
    description TEXT NOT NULL,
    initiator TEXT NOT NULL,
    source_ip TEXT NOT NULL,
    event_time TIMESTAMPTZ,
    raw_payload TEXT NOT NULL,
    status TEXT DEFAULT 'NEW',
    assigned_to BIGINT DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Создание таблицы инженеров
CREATE TABLE IF NOT EXISTS engineers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id BIGINT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    role TEXT DEFAULT 'engineer',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

-- Создание таблицы кодов приглашений
CREATE TABLE IF NOT EXISTS invite_codes (
    code TEXT PRIMARY KEY,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT FALSE
);

-- Создание таблицы для хранения ID сообщений в Telegram
CREATE TABLE IF NOT EXISTS incident_messages (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    message_id INT NOT NULL,
    PRIMARY KEY (incident_id, chat_id)
);

-- Вставка админа по умолчанию (замени на свой Telegram ID!)
INSERT INTO engineers (telegram_id, name, role, is_active) 
VALUES (1852630149, 'Sekku', 'admin', TRUE)
ON CONFLICT (telegram_id) DO NOTHING;