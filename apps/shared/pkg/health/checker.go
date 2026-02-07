package health

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Checker struct {
	client   *http.Client
	logger   *slog.Logger
	timeout  time.Duration
	retries  int
	interval time.Duration
}

func NewChecker(timeout time.Duration, retries int, interval time.Duration, logger *slog.Logger) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: 10 * time.Second,
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

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

func (h *Checker) CheckWithBackoff(ctx context.Context, url string) error {
	h.logger.Info("Starting health check with exponential backoff", "url", url)

	deadline := time.Now().Add(h.timeout)
	backoff := h.interval

	for attempt := 1; ; attempt++ {
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

		h.logger.Warn("Health check failed, backing off",
			"url", url,
			"attempt", attempt,
			"nextBackoff", backoff,
			"error", err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
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
