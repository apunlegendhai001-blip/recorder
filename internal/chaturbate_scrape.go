package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/HeapOfChaos/goondvr/server"
)

// unescapeUnicode converts \uXXXX sequences to actual characters
func unescapeUnicode(s string) string {
	re := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		hex := match[2:] // Remove \u prefix
		code, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return match
		}
		return string(rune(code))
	})
}

// ScrapeChaturbateStream scrapes the public Chaturbate page to get stream info
// This works without authentication and bypasses Cloudflare using FlareSolverr
func ScrapeChaturbateStream(ctx context.Context, username string) (string, string, error) {
	pageURL := fmt.Sprintf("%s%s/", server.Config.Domain, username)
	
	if server.Config.Debug {
		fmt.Printf("[DEBUG] Scraping page: %s\n", pageURL)
	}
	
	// Use FlareSolverr to get the page content
	cookies, userAgent, err := GetFreshCookies(ctx, pageURL)
	if err != nil {
		return "", "", fmt.Errorf("flaresolverr failed: %w", err)
	}
	
	// Update global config with fresh cookies and user agent
	server.Config.Cookies = cookies
	server.Config.UserAgent = userAgent
	
	// Now get the actual page content with the cookies
	req := NewReq()
	body, err := req.Get(ctx, pageURL)
	if err != nil {
		return "", "", fmt.Errorf("get page: %w", err)
	}
	
	if server.Config.Debug {
		fmt.Printf("[DEBUG] Page content length: %d bytes\n", len(body))
	}
	
	// Extract HLS URL from the page
	// Look for m3u8 URL in the page source
	m3u8Regex := regexp.MustCompile(`https://[^"'\s]+\.m3u8[^"'\s]*`)
	matches := m3u8Regex.FindAllString(body, -1)
	
	if len(matches) == 0 {
		// Try to find it in embedded JSON
		jsonRegex := regexp.MustCompile(`"hls_source":\s*"([^"]+)"`)
		jsonMatches := jsonRegex.FindStringSubmatch(body)
		if len(jsonMatches) > 1 {
			hlsURL := strings.ReplaceAll(jsonMatches[1], `\/`, `/`)
			if server.Config.Debug {
				fmt.Printf("[DEBUG] Found HLS URL in JSON: %s\n", hlsURL)
			}
			return hlsURL, "public", nil
		}
		
		// Check if offline
		if strings.Contains(body, "offline") || strings.Contains(body, "is not currently broadcasting") {
			return "", "offline", nil
		}
		
		// Check if private
		if strings.Contains(body, "private") || strings.Contains(body, "Private Show") {
			return "", "private", nil
		}
		
		return "", "", fmt.Errorf("no stream URL found on page")
	}
	
	// Use the first m3u8 URL found
	hlsURL := matches[0]
	
	if server.Config.Debug {
		fmt.Printf("[DEBUG] Found HLS URL: %s\n", hlsURL)
	}
	
	return hlsURL, "public", nil
}

// ScrapeChaturbateStreamWithFlareSolverr uses FlareSolverr to get the full page with JS execution
func ScrapeChaturbateStreamWithFlareSolverr(ctx context.Context, username string) (string, string, error) {
	pageURL := fmt.Sprintf("%s%s/", server.Config.Domain, username)
	
	resp, err := GetFlareSolverrResponse(ctx, pageURL)
	if err != nil {
		return "", "", fmt.Errorf("flaresolverr failed: %w", err)
	}
	
	body := resp.Solution.Response
	
	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr page content length: %d bytes\n", len(body))
	}
	
	// Extract cookies and user agent
	var cookieParts []string
	for _, cookie := range resp.Solution.Cookies {
		if cookie.Name == "cf_clearance" || cookie.Name == "csrftoken" {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
	}
	
	if len(cookieParts) > 0 {
		server.Config.Cookies = strings.Join(cookieParts, "; ")
		server.Config.UserAgent = resp.Solution.UserAgent
	}
	
	// Look for the embedded player data
	// Chaturbate embeds stream info in window.initialRoomDossier or similar
	patterns := []string{
		`"hls_source":\s*"([^"]+)"`,
		`"hlsSource":\s*"([^"]+)"`,
		`https://[^"'\s]+\.m3u8[^"'\s]*`,
	}
	
	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindStringSubmatch(body)
		if len(matches) > 1 {
			hlsURL := strings.ReplaceAll(matches[1], `\/`, `/`)
			// Unescape unicode characters like \u002D
			hlsURL = unescapeUnicode(hlsURL)
			if server.Config.Debug {
				fmt.Printf("[DEBUG] Found HLS URL with pattern %s: %s\n", pattern, hlsURL)
			}
			return hlsURL, "public", nil
		} else if len(matches) > 0 {
			hlsURL := matches[0]
			// Unescape unicode characters
			hlsURL = unescapeUnicode(hlsURL)
			if server.Config.Debug {
				fmt.Printf("[DEBUG] Found HLS URL: %s\n", hlsURL)
			}
			return hlsURL, "public", nil
		}
	}
	
	// Check room status
	if strings.Contains(body, "offline") || strings.Contains(body, "is not currently broadcasting") {
		return "", "offline", nil
	}
	
	if strings.Contains(body, "private") || strings.Contains(body, "Private Show") {
		return "", "private", nil
	}
	
	return "", "", fmt.Errorf("no stream URL found on page")
}

// GetFlareSolverrResponse gets the full response including HTML content
func GetFlareSolverrResponse(ctx context.Context, url string) (*FlareSolverrResponse, error) {
	flaresolverrURL := "http://localhost:8191/v1"
	
	reqBody := FlareSolverrRequest{
		Cmd:        "request.get",
		URL:        url,
		MaxTimeout: 90000, // 90 seconds for Cloudflare challenges
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	if server.Config.Debug {
		fmt.Printf("[DEBUG] FlareSolverr: requesting %s\n", url)
	}
	
	req := NewReq()
	body, err := req.PostJSON(ctx, flaresolverrURL, string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	
	var fsResp FlareSolverrResponse
	if err := json.Unmarshal([]byte(body), &fsResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	
	if fsResp.Status != "ok" {
		return nil, fmt.Errorf("flaresolverr error: %s", fsResp.Message)
	}
	
	return &fsResp, nil
}
