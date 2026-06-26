// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extservice

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStqlString(t *testing.T) {
	assert.Equal(t, `"a\"b\\c"`, stqlString(`a"b\c`))
}

func TestGetServiceSnapshot_EscapesServiceId(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := &StackStateHttpClient{Client: resty.New().SetBaseURL(srv.URL)}
	// A service id that tries to break out of the STQL string literal / JSON body.
	maliciousId := `1") OR (1=1`
	_, _, err := client.GetServiceSnapshot(context.Background(), maliciousId)
	require.NoError(t, err)

	// The request body must remain valid JSON despite the embedded quotes...
	var body struct {
		Query string `json:"query"`
	}
	require.NoError(t, json.Unmarshal(capturedBody, &body), "request body is not valid JSON: %s", capturedBody)

	// ...and the id stays inside a single, escaped STQL string literal (no breakout).
	assert.Equal(t, `(id = "1\") OR (1=1")`, body.Query)
}
