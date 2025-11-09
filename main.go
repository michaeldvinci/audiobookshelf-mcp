package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func getEnvOrParam(paramValue, envKey string) string {
	if paramValue != "" {
		return paramValue
	}
	return os.Getenv(envKey)
}

func getABSConfig(request mcp.CallToolRequest) (baseURL, token string, err error) {
	baseURLParam := request.GetString("base_url", "")
	tokenParam := request.GetString("token", "")

	baseURL = getEnvOrParam(baseURLParam, "ABS_BASE_URL")
	token = getEnvOrParam(tokenParam, "ABS_API_KEY")

	if baseURL == "" {
		return "", "", fmt.Errorf("base_url parameter or ABS_BASE_URL environment variable is required")
	}
	if token == "" {
		return "", "", fmt.Errorf("token parameter or ABS_API_KEY environment variable is required")
	}

	// Always append /api to the base URL
	baseURL = fmt.Sprintf("%s/api", baseURL)

	return baseURL, token, nil
}

// Helper to add base_url and token parameters to a tool
func withABSAuth() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("base_url",
			mcp.Description("Audiobookshelf base API URL, e.g. https://abs.example.com/api (defaults to ABS_BASE_URL env var)"),
		),
		mcp.WithString("token",
			mcp.Description("Bearer token used to authenticate with Audiobookshelf (defaults to ABS_API_KEY env var)"),
		),
	}
}

// Helper to create a simple list/get tool pair
func createSimpleGETHandler(path string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURL, token, err := getABSConfig(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body, err := absGET(ctx, baseURL, token, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(body)), nil
	}
}

// Helper to create a GET handler with an ID parameter
func createGETByIDHandler(pathTemplate, idParamName string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURL, token, err := getABSConfig(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		id, err := request.RequireString(idParamName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body, err := absGET(ctx, baseURL, token, fmt.Sprintf(pathTemplate, id))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(body)), nil
	}
}

// Helper to create a GET handler with ID and optional sub-resource
func createGETByIDWithSubResourceHandler(basePath, idParamName string, subResources []string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURL, token, err := getABSConfig(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		id, err := request.RequireString(idParamName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Build the path - check if any sub-resource is requested
		path := fmt.Sprintf(basePath, id)
		for _, subResource := range subResources {
			if request.GetBool(subResource, false) {
				path = fmt.Sprintf("%s/%s", path, subResource)
				break // Only one sub-resource at a time
			}
		}

		body, err := absGET(ctx, baseURL, token, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(body)), nil
	}
}

func absGET(ctx context.Context, baseURL, token, path string) ([]byte, error) {
	fullURL := strings.TrimSuffix(baseURL, "/") + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ABS API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ABS API returned %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return body, nil
}

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Audiobookshelf MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	// Add ABS tools
	// Libraries tools
	librariesOpts := append(withABSAuth(), mcp.WithDescription("List Audiobookshelf libraries"))
	librariesTool := mcp.NewTool("libraries", librariesOpts...)

	libraryOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf library by ID, optionally with items or authors"),
		mcp.WithString("library_id", mcp.Required(), mcp.Description("Library identifier to fetch")),
		mcp.WithBoolean("items", mcp.Description("Include all items in the library")),
		mcp.WithBoolean("authors", mcp.Description("Include all authors in the library")),
	)
	libraryTool := mcp.NewTool("library", libraryOpts...)

	// Items tools
	itemOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf item (audiobook or podcast) by ID"),
		mcp.WithString("item_id", mcp.Required(), mcp.Description("Item identifier to fetch")),
	)
	itemTool := mcp.NewTool("item", itemOpts...)

	// Authors tools
	authorOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf author by ID"),
		mcp.WithString("author_id", mcp.Required(), mcp.Description("Author identifier to fetch")),
	)
	authorTool := mcp.NewTool("author", authorOpts...)

	// User tools
	meOpts := append(withABSAuth(), mcp.WithDescription("Get authenticated user information"))
	meTool := mcp.NewTool("me", meOpts...)

	// Collections tools
	collectionsOpts := append(withABSAuth(), mcp.WithDescription("List all Audiobookshelf collections"))
	collectionsTool := mcp.NewTool("collections", collectionsOpts...)

	collectionOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf collection by ID"),
		mcp.WithString("collection_id", mcp.Required(), mcp.Description("Collection identifier to fetch")),
	)
	collectionTool := mcp.NewTool("collection", collectionOpts...)

	// Playlists tools
	playlistsOpts := append(withABSAuth(), mcp.WithDescription("List all Audiobookshelf playlists"))
	playlistsTool := mcp.NewTool("playlists", playlistsOpts...)

	playlistOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf playlist by ID"),
		mcp.WithString("playlist_id", mcp.Required(), mcp.Description("Playlist identifier to fetch")),
	)
	playlistTool := mcp.NewTool("playlist", playlistOpts...)

	// Add ABS Libraries handlers
	s.AddTool(librariesTool, createSimpleGETHandler("/libraries"))
	s.AddTool(libraryTool, createGETByIDWithSubResourceHandler("/libraries/%s", "library_id", []string{"items", "authors"}))

	// Add ABS Items handlers
	s.AddTool(itemTool, createGETByIDHandler("/items/%s", "item_id"))

	// Add ABS Authors handlers
	s.AddTool(authorTool, createGETByIDHandler("/authors/%s", "author_id"))

	// Add ABS Me handler
	s.AddTool(meTool, createSimpleGETHandler("/me"))

	// Add ABS Collections handlers
	s.AddTool(collectionsTool, createSimpleGETHandler("/collections"))
	s.AddTool(collectionTool, createGETByIDHandler("/collections/%s", "collection_id"))

	// Add ABS Playlists handlers
	s.AddTool(playlistsTool, createSimpleGETHandler("/playlists"))
	s.AddTool(playlistTool, createGETByIDHandler("/playlists/%s", "playlist_id"))

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
