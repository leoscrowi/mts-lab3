package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Logf func(format string, args ...any)

type Client struct {
	BaseURL string
	Headers map[string]string
	HTTP    *http.Client
	Log     Logf
}

func New(baseURL string, timeout time.Duration, logf Logf) *Client {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Headers: map[string]string{"Accept": "application/json"},
		HTTP:    &http.Client{Timeout: timeout, Transport: transport},
		Log:     logf,
	}
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
}

func (r *Response) JSON(v any) error {
	dec := json.NewDecoder(bytes.NewReader(r.Body))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode JSON: %w; body: %s", err, truncate(string(r.Body), 500))
	}
	return nil
}

func (r *Response) Map() (map[string]any, error) {
	var m map[string]any
	err := r.JSON(&m)
	return m, err
}

func (c *Client) Do(ctx context.Context, method, path string, body any, extra map[string]string) (*Response, error) {
	var reqBody []byte
	if body != nil {
		var err error
		if reqBody, err = json.Marshal(body); err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}

	c.Log("method, url: %s %s", method, req.URL)
	c.Log("headers: %s", formatHeaders(req.Header))
	if len(reqBody) > 0 {
		c.Log("body: %s", truncate(string(reqBody), 2000))
	}

	start := time.Now()
	resp, err := c.HTTP.Do(req)
	if err != nil {
		c.Log("method, url: %s %s: transport error: %v", method, req.URL, err)
		return nil, err
	}
	defer resp.Body.Close()

	c.Log("RESPONSE:")
	c.Log("%d %s in %s", resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start).Round(time.Millisecond))
	c.Log("content-type: %s", resp.Header.Get("Content-Type"))

	respBody, err := io.ReadAll(resp.Body)
	dur := time.Since(start)
	if err != nil {
		c.Log("read body failed after %s: %v", dur.Round(time.Millisecond), err)
		return nil, fmt.Errorf("read body: %w", err)
	}

	c.Log("%s: %s", dur.Round(time.Millisecond), truncate(string(respBody), 2000))

	return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: respBody, Duration: dur}, nil
}

func formatHeaders(h http.Header) string {
	parts := make([]string, 0, len(h))
	for k, vs := range h {
		v := strings.Join(vs, ",")
		if strings.EqualFold(k, "Authorization") {
			v = MaskToken(v)
		}
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

func MaskToken(s string) string {
	if len(s) <= 20 {
		return "***"
	}
	return s[:12] + "…" + s[len(s)-4:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + fmt.Sprintf("...(+%d bytes)", len(s)-n)
}
