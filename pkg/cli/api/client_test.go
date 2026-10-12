// Copyright (c) 2026 Probo Inc <hello@probo.com>.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/cli/api"
)

func TestDoUpload_FilePartContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{name: "detected image type", contentType: "image/png", want: "image/png"},
		{name: "empty type", contentType: "", want: "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reader, err := r.MultipartReader()
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				for {
					part, err := reader.NextPart()
					if err == io.EOF {
						break
					}

					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}

					if part.FormName() == "0" {
						got = part.Header.Get("Content-Type")
					}

					_, _ = io.Copy(io.Discard, part)

					_ = part.Close()
				}

				w.Header().Set("Content-Type", "application/json")

				_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
			}))
			t.Cleanup(server.Close)

			client := api.NewClient(server.URL, "token", "/graphql", time.Second)

			_, err := client.DoUpload(
				"mutation { ok }",
				map[string]any{},
				"variables.input.file",
				"diagram.png",
				tt.contentType,
				strings.NewReader("png"),
			)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDoUpload_RejectsHeaderLineBreak(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filename    string
		contentType string
	}{
		{name: "content type carriage return", filename: "diagram.png", contentType: "image/png\r\nX-Injected: 1"},
		{name: "content type line feed", filename: "diagram.png", contentType: "image/png\nX-Injected: 1"},
		{name: "filename line break", filename: "diagram.png\r\nX-Injected: 1", contentType: "image/png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var body string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				body = string(raw)

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
			}))
			t.Cleanup(server.Close)

			client := api.NewClient(server.URL, "token", "/graphql", time.Second)

			_, err := client.DoUpload(
				"mutation { ok }",
				map[string]any{},
				"variables.input.file",
				tt.filename,
				tt.contentType,
				strings.NewReader("png"),
			)
			require.Error(t, err)
			assert.ErrorContains(t, err, "line break")
			assert.NotContains(t, body, "X-Injected")
		})
	}
}
