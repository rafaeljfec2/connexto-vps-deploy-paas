package health

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var errHealthDeadline = errors.New("health check deadline reached")

type Checker struct {
	client    *http.Client
	tlsClient *http.Client
	logger    *slog.Logger
	timeout   time.Duration
	retries   int
	interval  time.Duration
}

func NewChecker(timeout time.Duration, retries int, interval time.Duration, logger *slog.Logger) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		tlsClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: newHealthCheckTLSConfig(),
			},
		},
		logger:   logger,
		timeout:  timeout,
		retries:  retries,
		interval: interval,
	}
}

func (h *Checker) Check(ctx context.Context, url string) error {
	h.logger.Info("Starting health check", "url", url, "retries", h.retries, "timeout", h.timeout)

	deadline := time.Now().Add(h.timeout)

	for attempt := 1; attempt <= h.retries; attempt++ {
		if time.Now().After(deadline) {
			return fmt.Errorf("health check timed out after %v", h.timeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := h.singleCheck(ctx, url)
		if err == nil {
			h.logger.Info("Health check passed", "url", url, "attempt", attempt)
			return nil
		}

		h.logger.Warn("Health check failed, retrying",
			"url", url,
			"attempt", attempt,
			"maxRetries", h.retries,
			"error", err,
		)

		if attempt < h.retries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(h.interval):
			}
		}
	}

	return fmt.Errorf("health check failed after %d attempts", h.retries)
}

func (h *Checker) singleCheck(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := h.client
	if strings.HasPrefix(url, "https://") {
		client = h.tlsClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

// CheckHooks lets a deploy loop resolve the target each attempt, abort when
// the container has already crashed, and publish probe failures to the deploy log.
type CheckHooks struct {
	ResolveURL    func(ctx context.Context) (string, error)
	BeforeAttempt func(ctx context.Context) error
	OnAttempt     func(attempt int, err error)
}

func (h *Checker) Timeout() time.Duration {
	return h.timeout
}

func (h *Checker) CheckWithBackoff(ctx context.Context, url string) error {
	return h.CheckUntil(ctx, time.Now().Add(h.timeout), CheckHooks{
		ResolveURL: func(context.Context) (string, error) {
			return url, nil
		},
	})
}

func (h *Checker) CheckUntil(ctx context.Context, deadline time.Time, hooks CheckHooks) error {
	if hooks.ResolveURL == nil {
		return fmt.Errorf("health check: resolve URL is required")
	}

	window := time.Until(deadline)
	if window < 0 {
		window = 0
	}
	backoff := h.interval
	var lastErr error

	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return healthTimeoutError(window, lastErr)
		}
		if hooks.BeforeAttempt != nil {
			if err := hooks.BeforeAttempt(ctx); err != nil {
				return err
			}
		}

		target, err := hooks.ResolveURL(ctx)
		if err != nil {
			lastErr = err
			h.notifyAttempt(hooks, attempt, err)
			if waitErr := h.wait(ctx, deadline, backoff); waitErr != nil {
				return h.waitResult(waitErr, window, lastErr)
			}
			backoff = nextBackoff(backoff)
			continue
		}

		err = h.singleCheck(ctx, target)
		if err == nil {
			h.logger.Info("Health check passed", "url", target, "attempt", attempt)
			return nil
		}

		lastErr = err
		h.notifyAttempt(hooks, attempt, err)
		h.logger.Warn("Health check failed, backing off",
			"url", target,
			"attempt", attempt,
			"nextBackoff", backoff,
			"error", err,
		)
		if waitErr := h.wait(ctx, deadline, backoff); waitErr != nil {
			return h.waitResult(waitErr, window, lastErr)
		}
		backoff = nextBackoff(backoff)
	}
}

func (h *Checker) notifyAttempt(hooks CheckHooks, attempt int, err error) {
	if hooks.OnAttempt != nil {
		hooks.OnAttempt(attempt, err)
	}
}

func (h *Checker) wait(ctx context.Context, deadline time.Time, backoff time.Duration) error {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return errHealthDeadline
	}
	if backoff > remaining {
		backoff = remaining
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (h *Checker) waitResult(waitErr error, window time.Duration, lastErr error) error {
	if errors.Is(waitErr, errHealthDeadline) {
		return healthTimeoutError(window, lastErr)
	}
	return waitErr
}

func healthTimeoutError(window time.Duration, lastErr error) error {
	if lastErr == nil {
		return fmt.Errorf("health check timed out after %v", window)
	}
	return fmt.Errorf("health check timed out after %v: %w", window, lastErr)
}

func nextBackoff(current time.Duration) time.Duration {
	if current <= 0 {
		return time.Second
	}
	next := current * 2
	if next > 30*time.Second {
		return 30 * time.Second
	}
	return next
}

func newHealthCheckTLSConfig() *tls.Config {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("health check: no TLS certificate presented")
			}
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("health check: invalid certificate: %w", err)
			}
			now := time.Now()
			if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
				return fmt.Errorf("health check: certificate expired (notBefore=%v, notAfter=%v)", cert.NotBefore, cert.NotAfter)
			}
			return nil
		},
	}
}

func (h *Checker) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
}

func (h *Checker) SetRetries(retries int) {
	h.retries = retries
}

func (h *Checker) SetInterval(interval time.Duration) {
	h.interval = interval
}
