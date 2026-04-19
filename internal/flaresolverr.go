package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/HeapOfChaos/goondvr/server"
)

type FlareSolverrRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int    `json:"maxTimeout"`
}

type FlareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		URL      string `json:"url"`
		Status   int    `json:"status"`
		Response string `json:"response"` // HTML content
		Cookies []struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Expires  float64 `json:"expires"`
			Size     int     `json:"size"`
			HttpOnly bool    `json:"httpOnly"`
			Secure   bool    `json:"secure"`
			SameSite string  `json:"sameSite"`
		} `json:"cookies"`
		UserAgent string `json:"userAgent"`
	} `json:"solution"`
}

// GetFreshCookies uses FlareSolverr to bypass Cloudflare and get fresh cookies
func GetFreshCookies(ctx context.Context, url string) (string, string, error) {
	flaresolverrURL := os.Getenv("FLARESOLVERR_URL")
	if flaresolverrURL == "" {
		flaresolverrURL = "http://localhost:8191/v1"
	}

	reqBody := FlareSolverrRequest{
		Cmd:        "request.get",
		URL:        url,
		MaxTimeout: 90000, // 90 seconds for Cloudflare challenges
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr: requesting %s\n", url)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", flaresolverrURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second} // Increased timeout for Cloudflare challenges
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read body: %w", err)
	}

	var fsResp FlareSolverrResponse
	if err := json.Unmarshal(body, &fsResp); err != nil {
		return "", "", fmt.Errorf("unmarshal response: %w", err)
	}

	if fsResp.Status != "ok" {
		return "", "", fmt.Errorf("flaresolverr error: %s", fsResp.Message)
	}

	// Extract cookies
	var cookieParts []string
	for _, cookie := range fsResp.Solution.Cookies {
		if cookie.Name == "cf_clearance" || cookie.Name == "csrftoken" {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
	}

	if len(cookieParts) == 0 {
		return "", "", fmt.Errorf("no cookies found in response")
	}

	cookieStr := strings.Join(cookieParts, "; ")
	userAgent := fsResp.Solution.UserAgent

	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr: got %d cookies, user-agent: %s\n", len(cookieParts), userAgent)
	}

	return cookieStr, userAgent, nil
}
