package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostsChain(t *testing.T) {
	c := newREST(t)

	created := map[string]any{
		"title":   "test title",
		"body":    "test",
		"user_id": 1,
	}
	updatedTitle := "updated test title"

	var postID string

	ok := t.Run("POST /posts creates post", func(t *testing.T) {
		resp := do(t, c, http.MethodPost, "/posts", created, nil)
		requireStatus(t, resp, http.StatusCreated)
		m := requireJSON(t, resp)

		postID = requireID(t, m)
		assertField(t, m, "title", created["title"])
		assertField(t, m, "body", created["body"])
		assertField(t, m, "user_id", created["user_id"])
		t.Logf("created post id: %s", postID)
	})
	require.True(t, ok && postID != "", "request failed, id was not received - chain stopped")

	ok = t.Run("PATCH /posts/{id} updates title only", func(t *testing.T) {
		resp := do(t, c, http.MethodPatch, "/posts/"+postID, map[string]any{"title": updatedTitle}, nil)
		requireStatus(t, resp, http.StatusOK)
		m := requireJSON(t, resp)

		assertField(t, m, "id", postID)
		assertField(t, m, "title", updatedTitle)
		assertField(t, m, "body", created["body"])
	})
	require.True(t, ok, "PATCH failed")

	t.Run("GET /posts/{id} returns changes", func(t *testing.T) {
		resp := do(t, c, http.MethodGet, "/posts/"+postID, nil, nil)
		requireStatus(t, resp, http.StatusOK)
		m := requireJSON(t, resp)

		assertField(t, m, "id", postID)
		assertField(t, m, "title", updatedTitle)
		assertField(t, m, "body", created["body"])
		assertField(t, m, "user_id", created["user_id"])
	})
}
