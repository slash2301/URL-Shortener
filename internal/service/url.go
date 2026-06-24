package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"url-shortener/internal/cache"
	"url-shortener/internal/model"
	"url-shortener/internal/repository"
)

const base62Chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var ErrNotFound    = errors.New("short code not found")
var ErrExpired     = errors.New("short URL has expired")
var ErrAliasExists = errors.New("custom alias already taken")

type URLService struct {
	repo         *repository.URLRepository
	cache        *cache.RedisCache
	shortCodeLen int
	baseURL      string
}

func NewURLService(
	repo *repository.URLRepository,
	cache *cache.RedisCache,
	shortCodeLen int,
	baseURL string,
) *URLService {
	return &URLService{
		repo:         repo,
		cache:        cache,
		shortCodeLen: shortCodeLen,
		baseURL:      baseURL,
	}
}

// ── SHORTEN ───────────────────────────────────────────────────────

func (s *URLService) Shorten(ctx context.Context, req *model.ShortenRequest) (*model.ShortenResponse, error) {
	if err := validateURL(req.URL); err != nil {
		return nil, err
	}

	var shortCode string
	var err error

	if req.CustomAlias != nil && *req.CustomAlias != "" {
		exists, err := s.repo.ExistsByShortCode(ctx, *req.CustomAlias)
		if err != nil {
			return nil, fmt.Errorf("checking alias: %w", err)
		}
		if exists {
			return nil, ErrAliasExists
		}
		shortCode = *req.CustomAlias
	} else {
		shortCode, err = s.generateUniqueCode(ctx)
		if err != nil {
			return nil, err
		}
	}

	var expiresAt *time.Time
	if req.ExpiryDays != nil && *req.ExpiryDays > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiryDays) * 24 * time.Hour)
		expiresAt = &t
	}

	u := &model.URL{
		ID:          uuid.New(),
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		CustomAlias: req.CustomAlias,
		ExpiresAt:   expiresAt,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("saving URL: %w", err)
	}

	// Pre-warm cache after creation
	if err := s.cache.SetURL(ctx, u); err != nil {
		// Non-fatal — log and continue
		log.Warn().Err(err).Str("code", shortCode).Msg("Failed to pre-warm cache")
	}

	return &model.ShortenResponse{
		ShortCode:   u.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, u.ShortCode),
		OriginalURL: u.OriginalURL,
		ExpiresAt:   u.ExpiresAt,
		CreatedAt:   u.CreatedAt,
	}, nil
}

// ── RESOLVE — cache-aside pattern ────────────────────────────────

func (s *URLService) Resolve(ctx context.Context, code string) (*model.URL, error) {
	// 1. Check negative cache — known-missing codes
	if s.cache.IsNotFound(ctx, code) {
		return nil, ErrNotFound
	}

	// 2. Check Redis cache
	u, err := s.cache.GetURL(ctx, code)
	if err == nil {
		// Cache HIT — check expiry then return
		if u.ExpiresAt != nil && time.Now().After(*u.ExpiresAt) {
			// Expired — evict and continue to DB
			_ = s.cache.DeleteURL(ctx, code)
		} else {
			log.Debug().Str("code", code).Msg("Cache HIT")
			go s.incrementClicksAsync(code)
			return u, nil
		}
	}

	// 3. Cache MISS — query PostgreSQL
	log.Debug().Str("code", code).Msg("Cache MISS")
	u, err = s.repo.GetByShortCode(ctx, code)
	if err != nil {
		// Negative cache — prevents DB hammering for bad codes
		_ = s.cache.SetNotFound(ctx, code)
		return nil, ErrNotFound
	}

	// Check expiry
	if u.ExpiresAt != nil && time.Now().After(*u.ExpiresAt) {
		return nil, ErrExpired
	}

	// 4. Write to cache for next request
	if err := s.cache.SetURL(ctx, u); err != nil {
		log.Warn().Err(err).Str("code", code).Msg("Failed to cache URL")
	}

	go s.incrementClicksAsync(code)
	return u, nil
}

// ── ANALYTICS ────────────────────────────────────────────────────

func (s *URLService) GetAnalytics(ctx context.Context, code string) (*model.AnalyticsResponse, error) {
	return s.repo.GetAnalytics(ctx, code)
}

// ── HELPERS ───────────────────────────────────────────────────────

func (s *URLService) incrementClicksAsync(code string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.repo.IncrementClicks(ctx, code); err != nil {
		log.Error().Err(err).Str("code", code).Msg("Failed to increment clicks")
	}
}

func (s *URLService) generateUniqueCode(ctx context.Context) (string, error) {
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		code := generateBase62(s.shortCodeLen)
		exists, err := s.repo.ExistsByShortCode(ctx, code)
		if err != nil {
			return "", fmt.Errorf("checking uniqueness: %w", err)
		}
		if !exists {
			return code, nil
		}
		log.Warn().Str("code", code).Int("attempt", i+1).Msg("Collision, retrying")
	}
	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxRetries)
}

func generateBase62(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = base62Chars[rand.Intn(len(base62Chars))]
	}
	return string(b)
}

func validateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL cannot be empty")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return errors.New("URL must start with http:// or https://")
	}
	if len(rawURL) > 2048 {
		return errors.New("URL too long (max 2048 characters)")
	}
	return nil
}