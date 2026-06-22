package model

import (
    "time"
    "github.com/google/uuid"
)

type URL struct {
    ID          uuid.UUID  `json:"id"`
    ShortCode   string     `json:"short_code"`
    OriginalURL string     `json:"original_url"`
    CustomAlias *string    `json:"custom_alias,omitempty"`
    Clicks      int64      `json:"clicks"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

type ClickEvent struct {
    ID          uuid.UUID `json:"id"`
    URLID       uuid.UUID `json:"url_id"`
    ShortCode   string    `json:"short_code"`
    IPAddress   string    `json:"ip_address"`
    UserAgent   string    `json:"user_agent"`
    Referer     string    `json:"referer"`
    CountryCode string    `json:"country_code"`
    CountryName string    `json:"country_name"`
    City        string    `json:"city"`
    DeviceType  string    `json:"device_type"`
    OS          string    `json:"os"`
    Browser     string    `json:"browser"`
    ClickedAt   time.Time `json:"clicked_at"`
}

// Request / Response DTOs

type ShortenRequest struct {
    URL         string  `json:"url"`
    CustomAlias *string `json:"custom_alias,omitempty"`
    ExpiryDays  *int    `json:"expiry_days,omitempty"`
}

type ShortenResponse struct {
    ShortCode   string     `json:"short_code"`
    ShortURL    string     `json:"short_url"`
    OriginalURL string     `json:"original_url"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}

type AnalyticsResponse struct {
    ShortCode   string         `json:"short_code"`
    OriginalURL string         `json:"original_url"`
    TotalClicks int64          `json:"total_clicks"`
    Countries   []CountryStat  `json:"top_countries"`
    Devices     []DeviceStat   `json:"devices"`
    DailyClicks []DailyStat    `json:"daily_clicks"`
}

type CountryStat struct {
    CountryCode string `json:"country_code"`
    CountryName string `json:"country_name"`
    Clicks      int64  `json:"clicks"`
}

type DeviceStat struct {
    DeviceType string `json:"device_type"`
    Clicks     int64  `json:"clicks"`
}

type DailyStat struct {
    Date   string `json:"date"`
    Clicks int64  `json:"clicks"`
}