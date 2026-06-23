package repository

import (
	"context"
	"time"

	//"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"url-shortener/internal/model"
)

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{db: db}
}

// ── URL CRUD ──────────────────────────────────────────────────────

func (r *URLRepository) Create(ctx context.Context, u *model.URL) error {
	query := `
		INSERT INTO urls (id, short_code, original_url, custom_alias, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	return r.db.QueryRow(ctx, query,
		u.ID, u.ShortCode, u.OriginalURL, u.CustomAlias, u.ExpiresAt,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *URLRepository) GetByShortCode(ctx context.Context, code string) (*model.URL, error) {
	query := `
		SELECT id, short_code, original_url, custom_alias, clicks, expires_at, created_at, updated_at
		FROM urls
		WHERE short_code = $1`

	u := &model.URL{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&u.ID, &u.ShortCode, &u.OriginalURL, &u.CustomAlias,
		&u.Clicks, &u.ExpiresAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *URLRepository) ExistsByShortCode(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`, code,
	).Scan(&exists)
	return exists, err
}

func (r *URLRepository) IncrementClicks(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE urls SET clicks = clicks + 1, updated_at = NOW() WHERE short_code = $1`, code,
	)
	return err
}

// ── ANALYTICS ────────────────────────────────────────────────────

func (r *URLRepository) SaveClickEvent(ctx context.Context, e *model.ClickEvent) error {
	query := `
		INSERT INTO click_events
			(id, url_id, short_code, ip_address, user_agent, referer,
			 country_code, country_name, city, device_type, os, browser, clicked_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`

	_, err := r.db.Exec(ctx, query,
		e.ID, e.URLID, e.ShortCode, e.IPAddress, e.UserAgent, e.Referer,
		e.CountryCode, e.CountryName, e.City, e.DeviceType, e.OS, e.Browser, e.ClickedAt,
	)
	return err
}

func (r *URLRepository) GetAnalytics(ctx context.Context, code string) (*model.AnalyticsResponse, error) {
	u, err := r.GetByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}

	resp := &model.AnalyticsResponse{
		ShortCode:   u.ShortCode,
		OriginalURL: u.OriginalURL,
		TotalClicks: u.Clicks,
	}

	// Top countries
	rows, err := r.db.Query(ctx, `
		SELECT country_code, country_name, COUNT(*) as clicks
		FROM click_events
		WHERE short_code = $1
		GROUP BY country_code, country_name
		ORDER BY clicks DESC
		LIMIT 10`, code)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s model.CountryStat
			if err := rows.Scan(&s.CountryCode, &s.CountryName, &s.Clicks); err == nil {
				resp.Countries = append(resp.Countries, s)
			}
		}
	}

	// Device breakdown
	drows, err := r.db.Query(ctx, `
		SELECT device_type, COUNT(*) as clicks
		FROM click_events
		WHERE short_code = $1
		GROUP BY device_type
		ORDER BY clicks DESC`, code)
	if err == nil {
		defer drows.Close()
		for drows.Next() {
			var s model.DeviceStat
			if err := drows.Scan(&s.DeviceType, &s.Clicks); err == nil {
				resp.Devices = append(resp.Devices, s)
			}
		}
	}

	// Daily clicks — last 30 days
	dclicks, err := r.db.Query(ctx, `
		SELECT DATE(clicked_at)::TEXT, COUNT(*) as clicks
		FROM click_events
		WHERE short_code = $1
		  AND clicked_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(clicked_at)
		ORDER BY DATE(clicked_at)`, code)
	if err == nil {
		defer dclicks.Close()
		for dclicks.Next() {
			var s model.DailyStat
			if err := dclicks.Scan(&s.Date, &s.Clicks); err == nil {
				resp.DailyClicks = append(resp.DailyClicks, s)
			}
		}
	}

	return resp, nil
}

// ── CLEANUP ───────────────────────────────────────────────────────

func (r *URLRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.Exec(ctx,
		`DELETE FROM urls WHERE expires_at IS NOT NULL AND expires_at < $1`, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}