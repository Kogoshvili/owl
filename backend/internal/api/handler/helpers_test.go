package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "val"})
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	require.Equal(t, "val", body["data"].(map[string]any)["key"])
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad request")
	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	require.Equal(t, "bad request", body["error"])
}

func TestParsePathID(t *testing.T) {
	t.Run("valid ID", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test/42", nil)
		r.SetPathValue("id", "42")
		w := httptest.NewRecorder()
		id, ok := parsePathID(w, r, "id")
		require.True(t, ok)
		require.Equal(t, int64(42), id)
	})

	t.Run("invalid ID", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test/abc", nil)
		r.SetPathValue("id", "abc")
		w := httptest.NewRecorder()
		_, ok := parsePathID(w, r, "id")
		require.False(t, ok)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing ID", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/test/", nil)
		w := httptest.NewRecorder()
		_, ok := parsePathID(w, r, "id")
		require.False(t, ok)
	})
}

func TestServeMuxRouting(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, "ok")
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp2, err := http.Post(ts.URL+"/test", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp2.StatusCode)
}
