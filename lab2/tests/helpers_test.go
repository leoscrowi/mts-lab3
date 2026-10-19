package tests

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mts-testing-lab2/config"
	"github.com/mts-testing-lab2/internal/apiclient"
)

var (
	cfg         = config.LoadConfig()
	runIdentity = uniq()
)

func uniq() string { return rand.Text()[:8] }

func newREST(t *testing.T) *apiclient.Client {
	c := apiclient.New(cfg.RestBaseURL, cfg.Timeout, t.Logf)
	c.Headers["X-Playground-Identity"] = runIdentity
	return c
}

func newGraphQL(t *testing.T) *apiclient.Client {
	return apiclient.New(cfg.GraphQLURL, cfg.Timeout, t.Logf)
}

func do(t *testing.T, c *apiclient.Client, method, path string, body any, headers map[string]string) *apiclient.Response {
	t.Helper()
	resp, err := c.Do(t.Context(), method, path, body, headers)
	require.NoError(t, err, "%s %s", method, path)
	return resp
}

func requireStatus(t *testing.T, resp *apiclient.Response, want int) {
	t.Helper()
	require.Equal(t, want, resp.StatusCode, "unexpected HTTP status; body: %s", resp.Body)
}

func requireJSON(t *testing.T, resp *apiclient.Response) map[string]any {
	t.Helper()
	require.Contains(t, resp.Header.Get("Content-Type"), "json", "response must be json")
	m, err := resp.Map()
	require.NoError(t, err)
	return m
}

func assertField(t *testing.T, m map[string]any, key string, want any) {
	t.Helper()
	assert.Equal(t, fmt.Sprint(want), fmt.Sprint(m[key]), "field %q", key)
}

func requireID(t *testing.T, m map[string]any) string {
	t.Helper()
	require.NotEmpty(t, m["id"], "response has no id")
	return fmt.Sprint(m["id"])
}

func assertErrorBody(t *testing.T, resp *apiclient.Response) {
	t.Helper()
	m, err := resp.Map()
	require.NoError(t, err, "error body must be json")
	assert.True(t, m["error"] != nil || m["message"] != nil, "error body has no message: %s", resp.Body)
}
