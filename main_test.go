package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// Mock server that simulates Audiobookshelf API
func setupMockABSServer() *httptest.Server {
	mux := http.NewServeMux()

	// Server endpoints
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"success": "true"})
	})

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"healthy": true})
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"isInit": true,
			"language": "en-us",
		})
	})

	// Libraries endpoints
	mux.HandleFunc("/api/libraries", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"libraries": []map[string]string{
				{"id": "lib1", "name": "Audiobooks"},
				{"id": "lib2", "name": "Podcasts"},
			},
		})
	})

	mux.HandleFunc("/api/libraries/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/libraries/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Library ID required", http.StatusBadRequest)
			return
		}

		libraryID := parts[0]

		if len(parts) == 1 {
			// Base library info
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": libraryID,
				"name": "Test Library",
			})
			return
		}

		// Sub-resources
		subResource := parts[1]
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"libraryId": libraryID,
			"resource": subResource,
			"data": []interface{}{},
		})
	})

	// Items endpoints
	mux.HandleFunc("/api/items/", func(w http.ResponseWriter, r *http.Request) {
		itemID := strings.TrimPrefix(r.URL.Path, "/api/items/")
		parts := strings.Split(itemID, "/")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": parts[0],
			"type": "book",
		})
	})

	// Authors endpoints
	mux.HandleFunc("/api/authors/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/authors/")
		parts := strings.Split(path, "/")

		if len(parts) > 1 && parts[1] == "image" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake-image-data"))
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": parts[0],
			"name": "Test Author",
		})
	})

	// Series endpoints
	mux.HandleFunc("/api/series/", func(w http.ResponseWriter, r *http.Request) {
		seriesID := strings.TrimPrefix(r.URL.Path, "/api/series/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": seriesID,
			"name": "Test Series",
		})
	})

	// Users endpoints
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []map[string]string{
				{"id": "user1", "username": "admin"},
			},
		})
	})

	mux.HandleFunc("/api/users/online", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []map[string]string{},
		})
	})

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/users/")
		parts := strings.Split(path, "/")

		if len(parts) == 1 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": parts[0],
				"username": "testuser",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"userId": parts[0],
			"resource": parts[1],
		})
	})

	// Me endpoints
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "current-user",
			"username": "me",
		})
	})

	mux.HandleFunc("/api/me/listening-sessions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": []interface{}{},
		})
	})

	mux.HandleFunc("/api/me/listening-stats", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"totalTime": 0,
		})
	})

	mux.HandleFunc("/api/me/items-in-progress", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []interface{}{},
		})
	})

	mux.HandleFunc("/api/me/progress/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"progress": 0.5,
		})
	})

	// Sessions endpoints
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": []interface{}{},
		})
	})

	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": sessionID,
		})
	})

	// Podcasts endpoints
	mux.HandleFunc("/api/podcasts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"podcasts": []interface{}{},
		})
	})

	mux.HandleFunc("/api/podcasts/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "podcast1",
		})
	})

	// Collections endpoints
	mux.HandleFunc("/api/collections", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"collections": []interface{}{},
		})
	})

	mux.HandleFunc("/api/collections/", func(w http.ResponseWriter, r *http.Request) {
		collectionID := strings.TrimPrefix(r.URL.Path, "/api/collections/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": collectionID,
		})
	})

	// Playlists endpoints
	mux.HandleFunc("/api/playlists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"playlists": []interface{}{},
		})
	})

	mux.HandleFunc("/api/playlists/", func(w http.ResponseWriter, r *http.Request) {
		playlistID := strings.TrimPrefix(r.URL.Path, "/api/playlists/")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": playlistID,
		})
	})

	// Backups endpoint
	mux.HandleFunc("/api/backups", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backups": []interface{}{},
		})
	})

	// Filesystem endpoint
	mux.HandleFunc("/api/filesystem", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"directories": []string{"/audiobooks", "/podcasts"},
		})
	})

	// Authorize endpoint
	mux.HandleFunc("/api/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]string{"id": "user1"},
			"server": map[string]string{"version": "2.0.0"},
		})
	})

	// Tags endpoint
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tags": []string{"fiction", "non-fiction"},
		})
	})

	// Genres endpoint
	mux.HandleFunc("/api/genres", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"genres": []string{"Fantasy", "Science Fiction", "Mystery"},
		})
	})

	return httptest.NewServer(mux)
}

// Helper to create a basic request with auth
func makeRequest(params map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "",
			Arguments: params,
		},
	}
}

func TestGetEnvOrParam(t *testing.T) {
	tests := []struct {
		name      string
		paramValue string
		envKey    string
		envValue  string
		expected  string
	}{
		{
			name:      "prefer param over env",
			paramValue: "param_value",
			envKey:    "TEST_KEY",
			envValue:  "env_value",
			expected:  "param_value",
		},
		{
			name:      "use env when param empty",
			paramValue: "",
			envKey:    "TEST_KEY",
			envValue:  "env_value",
			expected:  "env_value",
		},
		{
			name:      "return empty when both empty",
			paramValue: "",
			envKey:    "TEST_KEY",
			envValue:  "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnvOrParam(tt.paramValue, tt.envKey)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetABSConfig(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		envBaseURL  string
		envToken    string
		expectError bool
		expectedURL string
	}{
		{
			name: "valid params",
			params: map[string]interface{}{
				"base_url": "https://abs.example.com",
				"token":    "test-token",
			},
			expectError: false,
			expectedURL: "https://abs.example.com/api",
		},
		{
			name: "use env vars",
			params: map[string]interface{}{},
			envBaseURL: "https://env.example.com",
			envToken: "env-token",
			expectError: false,
			expectedURL: "https://env.example.com/api",
		},
		{
			name: "missing base_url",
			params: map[string]interface{}{
				"token": "test-token",
			},
			expectError: true,
		},
		{
			name: "missing token",
			params: map[string]interface{}{
				"base_url": "https://abs.example.com",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envBaseURL != "" {
				os.Setenv("ABS_BASE_URL", tt.envBaseURL)
				defer os.Unsetenv("ABS_BASE_URL")
			}
			if tt.envToken != "" {
				os.Setenv("ABS_API_KEY", tt.envToken)
				defer os.Unsetenv("ABS_API_KEY")
			}

			request := makeRequest(tt.params)
			baseURL, token, err := getABSConfig(request)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if baseURL != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, baseURL)
				}
				if token == "" {
					t.Error("expected non-empty token")
				}
			}
		})
	}
}

func TestABSGET(t *testing.T) {
	mockServer := setupMockABSServer()
	defer mockServer.Close()

	tests := []struct {
		name        string
		path        string
		expectError bool
		checkBody   func([]byte) error
	}{
		{
			name: "successful GET request",
			path: "/ping",
			expectError: false,
			checkBody: func(body []byte) error {
				if !strings.Contains(string(body), "success") {
					return fmt.Errorf("expected 'success' in body, got: %s", string(body))
				}
				return nil
			},
		},
		{
			name: "libraries endpoint",
			path: "/api/libraries",
			expectError: false,
			checkBody: func(body []byte) error {
				if !strings.Contains(string(body), "libraries") {
					return fmt.Errorf("expected 'libraries' in body, got: %s", string(body))
				}
				return nil
			},
		},
		{
			name: "404 endpoint",
			path: "/api/nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := absGET(context.Background(), mockServer.URL, "test-token", tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkBody != nil {
					if err := tt.checkBody(body); err != nil {
						t.Error(err)
					}
				}
			}
		})
	}
}

func TestEndpointHandlers(t *testing.T) {
	mockServer := setupMockABSServer()
	defer mockServer.Close()

	// Remove /api from mock server URL since getABSConfig adds it
	baseURL := strings.TrimSuffix(mockServer.URL, "/api")

	tests := []struct {
		name        string
		handler     func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		params      map[string]interface{}
		expectError bool
		checkResult func(*mcp.CallToolResult) error
	}{
		{
			name: "libraries handler",
			handler: createSimpleGETHandler("/libraries"),
			params: map[string]interface{}{
				"base_url": baseURL,
				"token":    "test-token",
			},
			expectError: false,
			checkResult: func(result *mcp.CallToolResult) error {
				if len(result.Content) == 0 {
					return fmt.Errorf("expected content, got empty")
				}
				return nil
			},
		},
		{
			name: "library by ID handler",
			handler: createGETByIDHandler("/libraries/%s", "library_id"),
			params: map[string]interface{}{
				"base_url":   baseURL,
				"token":      "test-token",
				"library_id": "lib123",
			},
			expectError: false,
			checkResult: func(result *mcp.CallToolResult) error {
				if len(result.Content) == 0 {
					return fmt.Errorf("expected content, got empty")
				}
				return nil
			},
		},
		{
			name: "author handler",
			handler: createGETByIDHandler("/authors/%s", "author_id"),
			params: map[string]interface{}{
				"base_url":  baseURL,
				"token":     "test-token",
				"author_id": "author123",
			},
			expectError: false,
		},
		{
			name: "series handler",
			handler: createGETByIDHandler("/series/%s", "series_id"),
			params: map[string]interface{}{
				"base_url":  baseURL,
				"token":     "test-token",
				"series_id": "series123",
			},
			expectError: false,
		},
		{
			name: "users handler",
			handler: createSimpleGETHandler("/users"),
			params: map[string]interface{}{
				"base_url": baseURL,
				"token":    "test-token",
			},
			expectError: false,
		},
		{
			name: "tags handler",
			handler: createSimpleGETHandler("/tags"),
			params: map[string]interface{}{
				"base_url": baseURL,
				"token":    "test-token",
			},
			expectError: false,
		},
		{
			name: "genres handler",
			handler: createSimpleGETHandler("/genres"),
			params: map[string]interface{}{
				"base_url": baseURL,
				"token":    "test-token",
			},
			expectError: false,
		},
		{
			name: "missing required ID parameter",
			handler: createGETByIDHandler("/libraries/%s", "library_id"),
			params: map[string]interface{}{
				"base_url": baseURL,
				"token":    "test-token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := makeRequest(tt.params)
			result, err := tt.handler(context.Background(), request)

			if tt.expectError {
				if err != nil || (result != nil && result.IsError) {
					// Expected error
					return
				}
				t.Error("expected error, got success")
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected result, got nil")
				}
				if result.IsError {
					t.Errorf("result returned error: %v", result)
				}
				if tt.checkResult != nil {
					if err := tt.checkResult(result); err != nil {
						t.Error(err)
					}
				}
			}
		})
	}
}

func TestSubResourceHandlers(t *testing.T) {
	mockServer := setupMockABSServer()
	defer mockServer.Close()

	baseURL := strings.TrimSuffix(mockServer.URL, "/api")

	tests := []struct {
		name           string
		basePath       string
		idParamName    string
		subResources   []string
		params         map[string]interface{}
		expectedInPath string
	}{
		{
			name:         "library with items sub-resource",
			basePath:     "/libraries/%s",
			idParamName:  "library_id",
			subResources: []string{"items", "authors", "series"},
			params: map[string]interface{}{
				"base_url":   baseURL,
				"token":      "test-token",
				"library_id": "lib123",
				"items":      true,
			},
			expectedInPath: "items",
		},
		{
			name:         "library without sub-resource",
			basePath:     "/libraries/%s",
			idParamName:  "library_id",
			subResources: []string{"items", "authors"},
			params: map[string]interface{}{
				"base_url":   baseURL,
				"token":      "test-token",
				"library_id": "lib123",
			},
			expectedInPath: "lib123",
		},
		{
			name:         "user with listening-sessions",
			basePath:     "/users/%s",
			idParamName:  "user_id",
			subResources: []string{"listening-sessions", "listening-stats"},
			params: map[string]interface{}{
				"base_url":          baseURL,
				"token":             "test-token",
				"user_id":           "user123",
				"listening-sessions": true,
			},
			expectedInPath: "listening-sessions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createGETByIDWithSubResourceHandler(tt.basePath, tt.idParamName, tt.subResources)
			request := makeRequest(tt.params)
			result, err := handler(context.Background(), request)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result == nil {
				t.Error("expected result, got nil")
			}
			if result.IsError {
				t.Errorf("result returned error: %v", result)
			}

			// Check that result contains expected data
			if len(result.Content) == 0 {
				t.Error("expected content, got empty")
			}
		})
	}
}

func TestAuthorizationHeader(t *testing.T) {
	// Create a test server that checks the Authorization header
	called := false
	var receivedAuth string

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer testServer.Close()

	expectedToken := "test-bearer-token"
	_, err := absGET(context.Background(), testServer.URL, expectedToken, "/test")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("test server was not called")
	}

	expectedAuth := "Bearer " + expectedToken
	if receivedAuth != expectedAuth {
		t.Errorf("expected Authorization header %q, got %q", expectedAuth, receivedAuth)
	}
}

func TestContextCancellation(t *testing.T) {
	// Create a server that delays response
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer testServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := absGET(ctx, testServer.URL, "token", "/test")
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
}
