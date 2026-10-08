package router

import (
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

const (
	abuseLogCooldown = 15 * time.Minute
	abusePruneEvery  = 10 * time.Minute
	abuseMaxKeys     = 20000
	abuseMaxHits     = 1000
	abuseMaxLabels   = 10
)

type abuseRule struct {
	window    time.Duration
	threshold int
	label     string
}

type ipActivity struct {
	hits       []time.Time
	userAgents []string
	routes     []string
	firstSeen  time.Time
	lastLogged time.Time
}

type activityTracker struct {
	mu        sync.Mutex
	event     string
	message   string
	rules     []abuseRule
	retention time.Duration
	entries   map[string]*ipActivity
	lastPrune time.Time
	logger    *logger.SecurityLogger
}

func newActivityTracker(securityLogger *logger.SecurityLogger, event, message string, rules ...abuseRule) *activityTracker {
	if securityLogger == nil {
		return nil
	}

	var retention time.Duration
	for _, rule := range rules {
		retention = max(retention, rule.window)
	}

	return &activityTracker{
		event:     event,
		message:   message,
		rules:     rules,
		retention: retention,
		entries:   make(map[string]*ipActivity),
		lastPrune: time.Now(),
		logger:    securityLogger,
	}
}

func (t *activityTracker) record(r *http.Request, trustedProxies []netip.Prefix) {
	if t == nil {
		return
	}

	route := r.URL.Path
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil && routeCtx.RoutePattern() != "" {
		route = routeCtx.RoutePattern()
	}

	if event, ok := t.observe(security.ResolvedClientIP(r, trustedProxies), r.UserAgent(), r.Method+" "+route, time.Now()); ok {
		t.logger.Log(event)
	}
}

func (t *activityTracker) observe(ip, userAgent, route string, now time.Time) (logger.SecurityEvent, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if now.Sub(t.lastPrune) > abusePruneEvery || len(t.entries) >= abuseMaxKeys {
		t.pruneLocked(now)
	}

	activity, exists := t.entries[ip]
	if !exists {
		activity = &ipActivity{firstSeen: now}
		t.entries[ip] = activity
	}

	cutoff := now.Add(-t.retention)
	keep := 0
	for keep < len(activity.hits) && activity.hits[keep].Before(cutoff) {
		keep++
	}
	activity.hits = append(activity.hits[keep:], now)
	if len(activity.hits) > abuseMaxHits {
		activity.hits = activity.hits[len(activity.hits)-abuseMaxHits:]
	}
	if userAgent == "" {
		userAgent = "(empty)"
	}
	activity.userAgents = appendLabel(activity.userAgents, userAgent)
	activity.routes = appendLabel(activity.routes, route)

	counts := make(map[string]int, len(t.rules))
	var triggered []string
	for _, rule := range t.rules {
		count := countSince(activity.hits, now.Add(-rule.window))
		counts[rule.label] = count
		if count >= rule.threshold {
			triggered = append(triggered, fmt.Sprintf("%d/%s", rule.threshold, rule.label))
		}
	}

	if len(triggered) == 0 || now.Sub(activity.lastLogged) < abuseLogCooldown {
		return logger.SecurityEvent{}, false
	}
	activity.lastLogged = now

	return logger.SecurityEvent{
		Time:       now,
		Event:      t.event,
		Message:    t.message,
		Rules:      triggered,
		ClientIP:   ip,
		UserAgents: slices.Clone(activity.userAgents),
		Routes:     slices.Clone(activity.routes),
		Counts:     counts,
		FirstSeen:  activity.firstSeen,
	}, true
}

func (t *activityTracker) pruneLocked(now time.Time) {
	t.lastPrune = now
	cutoff := now.Add(-t.retention)
	for ip, activity := range t.entries {
		if len(activity.hits) == 0 || activity.hits[len(activity.hits)-1].Before(cutoff) {
			delete(t.entries, ip)
		}
	}
	for ip := range t.entries {
		if len(t.entries) < abuseMaxKeys {
			break
		}
		delete(t.entries, ip)
	}
}

func countSince(hits []time.Time, since time.Time) int {
	count := 0
	for i := len(hits) - 1; i >= 0 && !hits[i].Before(since); i-- {
		count++
	}
	return count
}

func appendLabel(labels []string, value string) []string {
	if value == "" || slices.Contains(labels, value) || len(labels) >= abuseMaxLabels {
		return labels
	}
	return append(labels, value)
}

type abuseMonitor struct {
	registrations  *activityTracker
	rateLimits     *activityTracker
	trustedProxies []netip.Prefix
}

func newAbuseMonitor(securityLogger *logger.SecurityLogger, trustedProxies []netip.Prefix) *abuseMonitor {
	return &abuseMonitor{
		registrations: newActivityTracker(securityLogger, "suspicious_registration", "Too many accounts created from one IP",
			abuseRule{window: time.Minute, threshold: 10, label: "1m"},
			abuseRule{window: 24 * time.Hour, threshold: 50, label: "24h"},
		),
		rateLimits: newActivityTracker(securityLogger, "frequent_rate_limit", "IP frequently hits rate limit",
			abuseRule{window: time.Hour, threshold: 10, label: "1h"},
		),
		trustedProxies: trustedProxies,
	}
}

func (m *abuseMonitor) rateLimited(r *http.Request) {
	if m == nil {
		return
	}
	m.rateLimits.record(r, m.trustedProxies)
}

func (m *abuseMonitor) watchRegistrations(next http.Handler) http.Handler {
	if m == nil || m.registrations == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		if ww.Status() == http.StatusCreated {
			m.registrations.record(r, m.trustedProxies)
		}
	})
}
