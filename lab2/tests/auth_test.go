package tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mts-testing-lab2/internal/apiclient"
)

func TestAuthTokenChain(t *testing.T) {
	c := newREST(t)
	// почему то пароль из коллекции не сработал, брал отсюда: https://playground.nileslabs.com/docs/studio -> JWT auth flow
	token := login(t, c, "Bret", "Password@123")

	resp := do(t, c, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	requireStatus(t, resp, http.StatusOK)
	m := requireJSON(t, resp)

	requireID(t, m)
	require.Equal(t, "bret", m["username"])
}

func TestAuthErrors(t *testing.T) {
	c := newREST(t)

	t.Run("/auth/me without token returns 401", func(t *testing.T) {
		resp := do(t, c, http.MethodGet, "/auth/me", nil, nil)
		requireStatus(t, resp, http.StatusUnauthorized)
		assertErrorBody(t, resp)
	})

	t.Run("/auth/me with invalid token returns 401", func(t *testing.T) {
		resp := do(t, c, http.MethodGet, "/auth/me", nil, map[string]string{
			"Authorization": "Bearer invalid token value",
		})
		requireStatus(t, resp, http.StatusUnauthorized)
		assertErrorBody(t, resp)
	})
}

func requireJWT(t *testing.T, token string) {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "access token must be a JWT")

	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err, "JWT header must be base64url")

	var hdr map[string]any
	require.NoError(t, json.Unmarshal(raw, &hdr), "JWT header must be json")
	require.Contains(t, hdr, "alg", "JWT header must contain alg")
}

func login(t *testing.T, c *apiclient.Client, username, password string) string {
	t.Helper()
	resp := do(t, c, http.MethodPost, "/auth/login",
		map[string]any{"username": username, "password": password, "token_ttl": 86400}, nil)
	requireStatus(t, resp, http.StatusOK)
	m := requireJSON(t, resp)

	token, _ := m["access_token"].(string)
	require.NotEmpty(t, token, "access token must be a non-empty string")
	requireJWT(t, token)
	t.Logf("access_token received: %s", apiclient.MaskToken(token))
	return token
}
