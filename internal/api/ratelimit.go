package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ipLimiter is a per-IP token bucket. Entries idle for an hour are pruned.
type ipLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPLimiter(rps rate.Limit, burst int) *ipLimiter {
	l := &ipLimiter{visitors: make(map[string]*visitor), rps: rps, burst: burst}
	go func() {
		for range time.Tick(10 * time.Minute) {
			l.mu.Lock()
			for ip, v := range l.visitors {
				if time.Since(v.lastSeen) > time.Hour {
					delete(l.visitors, ip)
				}
			}
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

// rateLimit returns middleware allowing rps requests/second (burst capacity)
// per client IP. RealIP middleware must run first.
func rateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	l := newIPLimiter(rate.Limit(rps), burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(r.RemoteAddr) {
				writeError(w, http.StatusTooManyRequests, "too many requests — slow down a moment")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
