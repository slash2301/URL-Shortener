CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE urls (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_code   VARCHAR(10)  NOT NULL UNIQUE,
    original_url TEXT         NOT NULL,
    custom_alias VARCHAR(50),
    user_id      UUID,
    clicks       BIGINT       NOT NULL DEFAULT 0,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_urls_short_code ON urls(short_code);
CREATE INDEX idx_urls_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;

CREATE TABLE click_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url_id       UUID         NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    short_code   VARCHAR(10)  NOT NULL,
    ip_address   INET,
    user_agent   TEXT,
    referer      TEXT,
    country_code VARCHAR(2),
    country_name VARCHAR(100),
    city         VARCHAR(100),
    device_type  VARCHAR(20),
    os           VARCHAR(50),
    browser      VARCHAR(50),
    clicked_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_click_events_url_id    ON click_events(url_id);
CREATE INDEX idx_click_events_clicked_at ON click_events(clicked_at);
CREATE INDEX idx_click_events_country   ON click_events(country_code);