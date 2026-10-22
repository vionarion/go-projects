// Package metrics exposes Prometheus metrics for the url-shortener API.
package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus collectors used by the service.
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	LinksCreated    prometheus.Counter
	Redirects       prometheus.Counter
	RateLimited     prometheus.Counter
}

// New registers the collectors on reg and returns the Metrics handle.
func New(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)
	return &Metrics{
		requestsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, route and status.",
		}, []string{"method", "route", "status"}),
		requestDuration: f.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by method and route.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		LinksCreated: f.NewCounter(prometheus.CounterOpts{
			Name: "shortener_links_created_total",
			Help: "Total short links created.",
		}),
		Redirects: f.NewCounter(prometheus.CounterOpts{
			Name: "shortener_redirects_total",
			Help: "Total redirects served for existing short links.",
		}),
		RateLimited: f.NewCounter(prometheus.CounterOpts{
			Name: "shortener_rate_limited_total",
			Help: "Total requests rejected by the per-client rate limit.",
		}),
	}
}

// Middleware times every request and records it under its route template
// (e.g. "/:code"), not the raw path, so short codes don't blow up cardinality.
func (m *Metrics) Middleware(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()

	route := c.Route().Path
	if route == "" {
		route = "unknown"
	}
	status := c.Response().StatusCode()
	m.requestsTotal.WithLabelValues(c.Method(), route, strconv.Itoa(status)).Inc()
	m.requestDuration.WithLabelValues(c.Method(), route).Observe(time.Since(start).Seconds())

	return err
}

// Handler adapts promhttp's handler for registry reg to a Fiber handler.
func Handler(reg *prometheus.Registry) fiber.Handler {
	return adaptor.HTTPHandler(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}
