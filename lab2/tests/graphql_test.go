package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mts-testing-lab2/internal/apiclient"
)

type gqlError struct {
	Message string `json:"message"`
}

type qlResponse[T any] struct {
	Data   *T         `json:"data"`
	Errors []gqlError `json:"errors"`
}

func gql[T any](t *testing.T, c *apiclient.Client, query string, vars map[string]any) (*apiclient.Response, qlResponse[T]) {
	t.Helper()
	body := map[string]any{"query": query}
	if vars != nil {
		body["variables"] = vars
	}
	resp := do(t, c, http.MethodPost, "", body, nil)
	var out qlResponse[T]
	require.NoError(t, resp.JSON(&out))
	return resp, out
}

func TestGraphQLQueryHello(t *testing.T) {
	c := newGraphQL(t)
	const q = `query Hello { hello(person: { name: "Leonid", age: 21 }) }`

	resp, out := gql[struct {
		Hello string `json:"hello"`
	}](t, c, q, nil)

	requireStatus(t, resp, http.StatusOK)
	require.Empty(t, out.Errors, "unexpected GraphQL errors")
	require.NotNil(t, out.Data, "data is null")
	assert.Contains(t, out.Data.Hello, "Leonid", "greeting must contain the name")
}

func TestGraphQLMutationCreatePerson(t *testing.T) {
	c := newGraphQL(t)
	const m = `mutation CreatePerson {
  createPerson(person: { name: "Student", age: 20 }) { id name age }
}`

	type person struct {
		ID   any    `json:"id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	resp, out := gql[struct {
		CreatePerson *person `json:"createPerson"`
	}](t, c, m, nil)

	requireStatus(t, resp, http.StatusOK)
	require.Empty(t, out.Errors, "unexpected GraphQL errors")
	require.NotNil(t, out.Data, "data is null")
	require.NotNil(t, out.Data.CreatePerson, "createPerson is null")

	p := out.Data.CreatePerson
	assert.NotEmpty(t, p.ID, "createPerson.id")
	assert.Equal(t, "Student", p.Name)
	assert.Equal(t, 20, p.Age)
}

func TestGraphQLInvalidField(t *testing.T) {
	c := newGraphQL(t)
	resp, out := gql[map[string]any](t, c, `query { negativeTest }`, nil)

	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, resp.StatusCode)
	require.NotEmpty(t, out.Errors, "expected GraphQL errors")
	assert.Contains(t, out.Errors[0].Message, "negativeTest", "error must name the field")
	if out.Data != nil {
		assert.Empty(t, *out.Data, "data must be empty on validation error")
	}
}

func TestGraphQLSubscriptionGreetingsSSE(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body, _ := json.Marshal(map[string]string{"query": "subscription Greetings { greetings }"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.GraphQLURL, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	t.Logf("POST %s, body: %s", cfg.GraphQLURL, body)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	t.Logf("%d, content-type: %s", resp.StatusCode, resp.Header.Get("Content-Type"))

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"),
		"Content-Type = %q, expected text/event-stream", resp.Header.Get("Content-Type"))

	var greetings []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		t.Logf("sse: %s", line)

		if line == "event: complete" {
			break
		}

		data, ok := strings.CutPrefix(line, "data:")
		if !ok || strings.TrimSpace(data) == "" {
			continue
		}

		var ev qlResponse[struct {
			Greetings string `json:"greetings"`
		}]

		require.NoError(t, json.Unmarshal([]byte(data), &ev), "SSE data must be JSON: %s", data)
		require.Empty(t, ev.Errors, "unexpected GraphQL errors in event")
		require.NotNil(t, ev.Data, "event without data: %s", data)
		require.NotEmpty(t, ev.Data.Greetings, "event without greetings: %s", data)
		greetings = append(greetings, ev.Data.Greetings)
	}
	require.NoError(t, sc.Err(), "failed to read SSE stream")
	require.NotEmpty(t, greetings, "no greetings events received")
	t.Logf("received greetings: %q", greetings)
}
