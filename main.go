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

// Helper to create a GET handler for root-level endpoints (without /api prefix)
func createRootGETHandler(path string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURLParam := request.GetString("base_url", "")
		tokenParam := request.GetString("token", "")

		baseURL := getEnvOrParam(baseURLParam, "ABS_BASE_URL")
		token := getEnvOrParam(tokenParam, "ABS_API_KEY")

		if baseURL == "" {
			return mcp.NewToolResultError("base_url parameter or ABS_BASE_URL environment variable is required"), nil
		}
		if token == "" {
			return mcp.NewToolResultError("token parameter or ABS_API_KEY environment variable is required"), nil
		}

		// Don't append /api for root-level endpoints
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
		mcp.WithDescription("Retrieve a single Audiobookshelf library by ID, optionally with sub-resources"),
		mcp.WithString("library_id", mcp.Required(), mcp.Description("Library identifier to fetch")),
		mcp.WithBoolean("items", mcp.Description("Include all items in the library")),
		mcp.WithBoolean("authors", mcp.Description("Include all authors in the library")),
		mcp.WithBoolean("series", mcp.Description("Include all series in the library")),
		mcp.WithBoolean("collections", mcp.Description("Include all collections in the library")),
		mcp.WithBoolean("playlists", mcp.Description("Include all playlists in the library")),
		mcp.WithBoolean("personalized", mcp.Description("Include personalized view for the library")),
		mcp.WithBoolean("filterdata", mcp.Description("Include filter data for the library")),
		mcp.WithBoolean("stats", mcp.Description("Include library statistics")),
		mcp.WithBoolean("search", mcp.Description("Search the library items")),
		mcp.WithBoolean("episode-downloads", mcp.Description("Include episode downloads for the library")),
		mcp.WithBoolean("recent-episodes", mcp.Description("Include recent episodes for the library")),
	)
	libraryTool := mcp.NewTool("library", libraryOpts...)

	// Items tools
	itemOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf item (audiobook or podcast) by ID, optionally with sub-resources"),
		mcp.WithString("item_id", mcp.Required(), mcp.Description("Item identifier to fetch")),
		mcp.WithBoolean("cover", mcp.Description("Include cover image for the item")),
		mcp.WithBoolean("tone-object", mcp.Description("Include tone object for the item")),
	)
	itemTool := mcp.NewTool("item", itemOpts...)

	// Authors tools
	authorOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single Audiobookshelf author by ID"),
		mcp.WithString("author_id", mcp.Required(), mcp.Description("Author identifier to fetch")),
	)
	authorTool := mcp.NewTool("author", authorOpts...)

	// User tools
	meOpts := append(withABSAuth(),
		mcp.WithDescription("Get authenticated user information, or fetch specific user sub-resources"),
		mcp.WithBoolean("listening-sessions", mcp.Description("Get listening sessions for the user")),
		mcp.WithBoolean("listening-stats", mcp.Description("Get listening statistics for the user")),
		mcp.WithBoolean("items-in-progress", mcp.Description("Get items currently in progress for the user")),
		mcp.WithString("progress_item_id", mcp.Description("Get progress for a specific library item ID")),
		mcp.WithString("progress_episode_id", mcp.Description("Get progress for a specific episode ID (requires progress_item_id)")),
	)
	meTool := mcp.NewTool("me", meOpts...)

	// Sessions tools
	sessionsOpts := append(withABSAuth(), mcp.WithDescription("List all playback sessions"))
	sessionsTool := mcp.NewTool("sessions", sessionsOpts...)

	sessionOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single playback session by ID"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier to fetch")),
	)
	sessionTool := mcp.NewTool("session", sessionOpts...)

	// Podcasts tools
	podcastsOpts := append(withABSAuth(),
		mcp.WithDescription("List all podcasts, or fetch podcast-related resources"),
		mcp.WithBoolean("feed", mcp.Description("Get podcast RSS feed")),
		mcp.WithBoolean("opml", mcp.Description("Get podcast OPML export")),
	)
	podcastsTool := mcp.NewTool("podcasts", podcastsOpts...)

	podcastOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single podcast by ID, or fetch podcast sub-resources"),
		mcp.WithString("podcast_id", mcp.Required(), mcp.Description("Podcast identifier to fetch")),
		mcp.WithBoolean("downloads", mcp.Description("Get downloads for the podcast")),
		mcp.WithBoolean("search-episode", mcp.Description("Search for episodes in the podcast")),
		mcp.WithString("episode_id", mcp.Description("Get a specific episode by ID")),
	)
	podcastTool := mcp.NewTool("podcast", podcastOpts...)

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

	// Server status/health tools
	pingOpts := append(withABSAuth(), mcp.WithDescription("Simple health check endpoint"))
	pingTool := mcp.NewTool("ping", pingOpts...)

	healthcheckOpts := append(withABSAuth(), mcp.WithDescription("Server health verification endpoint"))
	healthcheckTool := mcp.NewTool("healthcheck", healthcheckOpts...)

	statusOpts := append(withABSAuth(), mcp.WithDescription("Get server initialization status and configuration"))
	statusTool := mcp.NewTool("status", statusOpts...)

	// Users tools
	usersOpts := append(withABSAuth(), mcp.WithDescription("List all Audiobookshelf users"))
	usersTool := mcp.NewTool("users", usersOpts...)

	usersOnlineOpts := append(withABSAuth(), mcp.WithDescription("Get currently online users"))
	usersOnlineTool := mcp.NewTool("users_online", usersOnlineOpts...)

	userOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single user by ID, optionally with sub-resources"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User identifier to fetch")),
		mcp.WithBoolean("listening-sessions", mcp.Description("Get listening sessions for the user")),
		mcp.WithBoolean("listening-stats", mcp.Description("Get listening statistics for the user")),
	)
	userTool := mcp.NewTool("user", userOpts...)

	// Series tools
	seriesOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve a single series by ID"),
		mcp.WithString("series_id", mcp.Required(), mcp.Description("Series identifier to fetch")),
	)
	seriesTool := mcp.NewTool("series", seriesOpts...)

	// Author image tool
	authorImageOpts := append(withABSAuth(),
		mcp.WithDescription("Retrieve author image by ID"),
		mcp.WithString("author_id", mcp.Required(), mcp.Description("Author identifier")),
	)
	authorImageTool := mcp.NewTool("author_image", authorImageOpts...)

	// Backups tools
	backupsOpts := append(withABSAuth(), mcp.WithDescription("List all server backups"))
	backupsTool := mcp.NewTool("backups", backupsOpts...)

	// Filesystem tools
	filesystemOpts := append(withABSAuth(), mcp.WithDescription("List available filesystem paths"))
	filesystemTool := mcp.NewTool("filesystem", filesystemOpts...)

	// Authorize tools
	authorizeOpts := append(withABSAuth(), mcp.WithDescription("Get authorized user and server information"))
	authorizeTool := mcp.NewTool("authorize", authorizeOpts...)

	// Tags and Genres tools
	tagsOpts := append(withABSAuth(), mcp.WithDescription("Get all library tags"))
	tagsTool := mcp.NewTool("tags", tagsOpts...)

	genresOpts := append(withABSAuth(), mcp.WithDescription("Get all available genres"))
	genresTool := mcp.NewTool("genres", genresOpts...)

	// Add ABS Libraries handlers
	s.AddTool(librariesTool, createSimpleGETHandler("/libraries"))
	s.AddTool(libraryTool, createGETByIDWithSubResourceHandler("/libraries/%s", "library_id", []string{
		"items",
		"authors",
		"series",
		"collections",
		"playlists",
		"personalized",
		"filterdata",
		"stats",
		"search",
		"episode-downloads",
		"recent-episodes",
	}))

	// Add ABS Items handlers
	s.AddTool(itemTool, createGETByIDWithSubResourceHandler("/items/%s", "item_id", []string{
		"cover",
		"tone-object",
	}))

	// Add ABS Authors handlers
	s.AddTool(authorTool, createGETByIDHandler("/authors/%s", "author_id"))

	// Add ABS Me handler
	s.AddTool(meTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURL, token, err := getABSConfig(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := "/me"

		// Check for simple boolean sub-resources first
		if request.GetBool("listening-sessions", false) {
			path = "/me/listening-sessions"
		} else if request.GetBool("listening-stats", false) {
			path = "/me/listening-stats"
		} else if request.GetBool("items-in-progress", false) {
			path = "/me/items-in-progress"
		} else if progressItemID := request.GetString("progress_item_id", ""); progressItemID != "" {
			// Handle progress endpoints with IDs
			if progressEpisodeID := request.GetString("progress_episode_id", ""); progressEpisodeID != "" {
				path = fmt.Sprintf("/me/progress/%s/%s", progressItemID, progressEpisodeID)
			} else {
				path = fmt.Sprintf("/me/progress/%s", progressItemID)
			}
		}

		body, err := absGET(ctx, baseURL, token, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(body)), nil
	})

	// Add ABS Sessions handlers
	s.AddTool(sessionsTool, createSimpleGETHandler("/sessions"))
	s.AddTool(sessionTool, createGETByIDHandler("/sessions/%s", "session_id"))

	// Add ABS Podcasts handlers
	s.AddTool(podcastsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURL, token, err := getABSConfig(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := "/podcasts"

		if request.GetBool("feed", false) {
			path = "/podcasts/feed"
		} else if request.GetBool("opml", false) {
			path = "/podcasts/opml"
		}

		body, err := absGET(ctx, baseURL, token, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(body)), nil
	})

	s.AddTool(podcastTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		baseURL, token, err := getABSConfig(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		podcastID, err := request.RequireString("podcast_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := fmt.Sprintf("/podcasts/%s", podcastID)

		if request.GetBool("downloads", false) {
			path = fmt.Sprintf("/podcasts/%s/downloads", podcastID)
		} else if request.GetBool("search-episode", false) {
			path = fmt.Sprintf("/podcasts/%s/search-episode", podcastID)
		} else if episodeID := request.GetString("episode_id", ""); episodeID != "" {
			path = fmt.Sprintf("/podcasts/%s/episode/%s", podcastID, episodeID)
		}

		body, err := absGET(ctx, baseURL, token, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(body)), nil
	})

	// Add ABS Collections handlers
	s.AddTool(collectionsTool, createSimpleGETHandler("/collections"))
	s.AddTool(collectionTool, createGETByIDHandler("/collections/%s", "collection_id"))

	// Add ABS Playlists handlers
	s.AddTool(playlistsTool, createSimpleGETHandler("/playlists"))
	s.AddTool(playlistTool, createGETByIDHandler("/playlists/%s", "playlist_id"))

	// Add Server status/health handlers (these are at root level, not /api)
	s.AddTool(pingTool, createRootGETHandler("/ping"))
	s.AddTool(healthcheckTool, createRootGETHandler("/healthcheck"))
	s.AddTool(statusTool, createRootGETHandler("/status"))

	// Add Users handlers
	s.AddTool(usersTool, createSimpleGETHandler("/users"))
	s.AddTool(usersOnlineTool, createSimpleGETHandler("/users/online"))
	s.AddTool(userTool, createGETByIDWithSubResourceHandler("/users/%s", "user_id", []string{
		"listening-sessions",
		"listening-stats",
	}))

	// Add Series handler
	s.AddTool(seriesTool, createGETByIDHandler("/series/%s", "series_id"))

	// Add Author image handler
	s.AddTool(authorImageTool, createGETByIDHandler("/authors/%s/image", "author_id"))

	// Add Backups handler
	s.AddTool(backupsTool, createSimpleGETHandler("/backups"))

	// Add Filesystem handler
	s.AddTool(filesystemTool, createSimpleGETHandler("/filesystem"))

	// Add Authorize handler
	s.AddTool(authorizeTool, createSimpleGETHandler("/authorize"))

	// Add Tags and Genres handlers
	s.AddTool(tagsTool, createSimpleGETHandler("/tags"))
	s.AddTool(genresTool, createSimpleGETHandler("/genres"))

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
