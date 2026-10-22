// Package api contains the HTTP handlers of the URL shortener.
package api

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/vionarion/go-projects/url-shortener/internal/metrics"
	"github.com/vionarion/go-projects/url-shortener/internal/store"
)

// Config holds the settings that come from environment variables.
type Config struct {
	Domain     string        // public host of the service, e.g. "s.example.com"
	Quota      int           // shorten requests per client per window
	Window     time.Duration // rate-limit window
	DefaultTTL time.Duration // link lifetime when the client does not ask for one
	MaxTTL     time.Duration
	CodeLength int
	TrustProxy bool // take the client IP from X-Forwarded-For (behind Traefik/nginx)
}

type App struct {
	cfg     Config
	store   store.Store
	metrics *metrics.Metrics
}

func New(cfg Config, st store.Store) *fiber.App {
	reg := prometheus.NewRegistry()
	a := &App{cfg: cfg, store: st, metrics: metrics.New(reg)}
	fcfg := fiber.Config{AppName: "url-shortener"}
	if cfg.TrustProxy {
		fcfg.ProxyHeader = fiber.HeaderXForwardedFor
	}
	app := fiber.New(fcfg)
	app.Use(recover.New(), logger.New(), a.metrics.Middleware)
	app.Get("/health", a.health)
	app.Get("/livez", a.livez)
	app.Get("/metrics", metrics.Handler(reg))
	app.Post("/api/v1/shorten", a.shorten)
	app.Get("/:code", a.resolve)
	return app
}

type shortenRequest struct {
	URL         string `json:"url"`
	Code        string `json:"code"`         // optional custom short code
	ExpiryHours int    `json:"expiry_hours"` // optional, 0 = default
}

type shortenResponse struct {
	URL              string `json:"url"`
	ShortURL         string `json:"short_url"`
	ExpiresInHours   int    `json:"expires_in_hours"`
	RateRemaining    int    `json:"rate_remaining"`
	RateResetSeconds int    `json:"rate_reset_seconds"`
}

var codeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func errJSON(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// livez only says the process is alive. It must not depend on Redis: a liveness probe on
// /health restarted every pod during a Redis outage.
func (a *App) livez(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "alive"})
}

func (a *App) health(c *fiber.Ctx) error {
	if err := a.store.Ping(c.UserContext()); err != nil {
		return errJSON(c, fiber.StatusServiceUnavailable, "storage unavailable")
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (a *App) shorten(c *fiber.Ctx) error {
	ctx := c.UserContext()

	left, reset, err := a.store.Take(ctx, c.IP(), a.cfg.Quota, a.cfg.Window)
	if err != nil {
		return errJSON(c, fiber.StatusServiceUnavailable, "storage unavailable")
	}
	if left < 0 {
		a.metrics.RateLimited.Inc()
		c.Set(fiber.HeaderRetryAfter, strconv.Itoa(int(reset.Seconds())))
		return errJSON(c, fiber.StatusTooManyRequests, "rate limit exceeded")
	}

	var req shortenRequest
	if err := c.BodyParser(&req); err != nil {
		return errJSON(c, fiber.StatusBadRequest, "cannot parse JSON")
	}

	target, err := normalizeURL(req.URL)
	if err != nil {
		return errJSON(c, fiber.StatusBadRequest, err.Error())
	}
	if sameHost(target, a.cfg.Domain) {
		return errJSON(c, fiber.StatusBadRequest, "links to this service are not allowed")
	}

	ttl := a.cfg.DefaultTTL
	if req.ExpiryHours < 0 {
		return errJSON(c, fiber.StatusBadRequest, "expiry_hours must be positive")
	}
	if req.ExpiryHours > 0 {
		ttl = time.Duration(req.ExpiryHours) * time.Hour
	}
	if ttl > a.cfg.MaxTTL {
		ttl = a.cfg.MaxTTL
	}

	code := req.Code
	if code != "" {
		if !codeRe.MatchString(code) {
			return errJSON(c, fiber.StatusBadRequest, "code must be 3-32 characters: letters, digits, - or _")
		}
		ok, err := a.store.SaveLink(ctx, code, target, ttl)
		if err != nil {
			return errJSON(c, fiber.StatusServiceUnavailable, "storage unavailable")
		}
		if !ok {
			return errJSON(c, fiber.StatusConflict, "code is already taken")
		}
	} else {
		code, err = a.saveRandom(c, target, ttl)
		if err != nil {
			return errJSON(c, fiber.StatusServiceUnavailable, "storage unavailable")
		}
	}

	a.metrics.LinksCreated.Inc()
	return c.Status(fiber.StatusCreated).JSON(shortenResponse{
		URL:              target,
		ShortURL:         a.shortURL(c, code),
		ExpiresInHours:   int(ttl.Hours()),
		RateRemaining:    left,
		RateResetSeconds: int(reset.Seconds()),
	})
}

// saveRandom generates codes until one is free; collisions are rare, so a few tries are enough.
func (a *App) saveRandom(c *fiber.Ctx, target string, ttl time.Duration) (string, error) {
	for i := 0; i < 5; i++ {
		code, err := randomCode(a.cfg.CodeLength)
		if err != nil {
			return "", err
		}
		ok, err := a.store.SaveLink(c.UserContext(), code, target, ttl)
		if err != nil {
			return "", err
		}
		if ok {
			return code, nil
		}
	}
	return "", errors.New("no free code found")
}

func (a *App) resolve(c *fiber.Ctx) error {
	target, err := a.store.GetLink(c.UserContext(), c.Params("code"))
	if errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, "short link not found")
	}
	if err != nil {
		return errJSON(c, fiber.StatusServiceUnavailable, "storage unavailable")
	}
	a.metrics.Redirects.Inc()
	return c.Redirect(target, fiber.StatusFound)
}

func (a *App) shortURL(c *fiber.Ctx, code string) string {
	host := a.cfg.Domain
	if host == "" {
		host = c.Hostname()
	}
	return c.Protocol() + "://" + host + "/" + code
}

// normalizeURL adds https:// when the scheme is missing and accepts only http(s) links with a host.
func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || !strings.Contains(u.Hostname(), ".") {
		return "", errors.New("invalid url")
	}
	return u.String(), nil
}

func sameHost(target, domain string) bool {
	if domain == "" {
		return false
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	return host == strings.TrimPrefix(strings.ToLower(domain), "www.")
}

func randomCode(n int) (string, error) {
	b := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		k, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[k.Int64()]
	}
	return string(b), nil
}
