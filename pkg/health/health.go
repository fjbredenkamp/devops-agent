// Package health provides HTTP endpoint probing for service health checks.
package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Probe sends an HTTP GET to the given URL and returns a human-readable
// status summary including status code, latency, and a health verdict.
func Probe(ctx context.Context, url string, timeoutSeconds int) (string, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
		// Don't follow redirects automatically — the agent should know if a redirect happens
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "devops-agent/1.0")

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return fmt.Sprintf("UNREACHABLE — %v\nURL: %s\nLatency: %v", err, url, latency), nil
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 400
	verdict := "HEALTHY"
	if !healthy {
		verdict = "UNHEALTHY"
	}

	return fmt.Sprintf(`%s
URL:         %s
Status:      %d %s
Latency:     %v
Content-Type: %s`,
		verdict,
		url,
		resp.StatusCode,
		http.StatusText(resp.StatusCode),
		latency.Round(time.Millisecond),
		resp.Header.Get("Content-Type"),
	), nil
}
