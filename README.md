# Audiobookshelf MCP Server

A Model Context Protocol (MCP) server that provides tools to interact with your [Audiobookshelf](https://www.audiobookshelf.org/) instance. Access your libraries, audiobooks, podcasts, authors, collections, and playlists through AI assistants that support MCP.

## Features

- List and retrieve libraries with optional sub-resources (items, authors)
- Get individual items (audiobooks or podcasts)
- Browse authors and their works
- Access collections and playlists
- Retrieve user information

## Installation

### Prerequisites

- Go 1.21 or later
- An Audiobookshelf instance with API access
- An API token from your Audiobookshelf instance

### Build from Source

```bash
git clone <repository-url>
cd abs-mcp
go build
```

## Configuration

The MCP server requires two pieces of configuration:

1. **ABS_BASE_URL** - The base URL of your Audiobookshelf instance (e.g., `https://abs.example.com`)
2. **ABS_API_KEY** - Your Audiobookshelf API token

### Getting Your API Token

1. Log into your Audiobookshelf instance
2. Go to Settings → Users → Your User
3. Click "Generate API Token" or copy your existing token

## Usage

### Environment Variables

You can set the configuration using environment variables:

```bash
export ABS_BASE_URL="https://abs.example.com"
export ABS_API_KEY="your-api-token-here"
```

Alternatively, you can pass these as parameters when calling tools (see Tool Parameters below).

### Setting Up with Witsy

[Witsy](https://github.com/nbonamy/witsy) is a desktop AI assistant that supports MCP servers. Here's how to set it up:

1. Open Witsy and go to Settings (⚙️ icon)
2. Navigate to the MCP section
3. Add a new server with the following configuration:

![Witsy MCP Configuration](_res/witsy.png)

**Configuration details:**
- **Type**: `stdio`
- **Label**: `abs-mcp` (or any name you prefer)
- **Command**: `/path/to/abs-mcp` (use the "Pick" button to select your compiled binary)
- **Arguments**: (leave empty)
- **Working Directory**: Any directory (e.g., `/Users/yourname/Downloads`)
- **Environment Variables**:
  - `ABS_BASE_URL` = `https://example.library.abs` (your Audiobookshelf URL)
  - `ABS_API_KEY` = `IM_A_LONG_STRING` (your API token)

4. Click "Save" to add the server
5. The server will now be available in your Witsy conversations!

### Setting Up with Claude Desktop

Add this to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "audiobookshelf": {
      "command": "/path/to/abs-mcp",
      "env": {
        "ABS_BASE_URL": "https://abs.example.com",
        "ABS_API_KEY": "your-api-token-here"
      }
    }
  }
}
```

## Available Tools

### Libraries

- **libraries** - List all libraries
- **library** - Get a single library by ID
  - Optional: `items=true` - Include all items in the library
  - Optional: `authors=true` - Include all authors in the library

### Items

- **item** - Get a single item (audiobook or podcast) by ID

### Authors

- **author** - Get a single author by ID

### Collections

- **collections** - List all collections
- **collection** - Get a single collection by ID

### Playlists

- **playlists** - List all playlists
- **playlist** - Get a single playlist by ID

### User

- **me** - Get authenticated user information

## Tool Parameters

All tools accept optional `base_url` and `token` parameters that override the environment variables:

```json
{
  "base_url": "https://abs.example.com",
  "token": "your-api-token-here"
}
```

This is useful if you need to access multiple Audiobookshelf instances or prefer not to use environment variables.

## Example Queries

Once configured, you can ask your AI assistant questions like:

- "Show me all my audiobook libraries"
- "What items are in my Fiction library?"
- "Get details about the author with ID abc123"
- "List all my playlists"
- "What's in my Currently Reading collection?"

The AI assistant will use the appropriate MCP tools to fetch the information from your Audiobookshelf instance.

## Development

### Project Structure

- `main.go` - Main server implementation
- Helper functions for API authentication and request handling
- MCP tool definitions and handlers

### Adding New Tools

To add a new tool:

1. Define the tool options using `mcp.NewTool()`
2. Add authentication parameters with `withABSAuth()`
3. Register the tool with `s.AddTool()`
4. Use helper functions like `createSimpleGETHandler()` or `createGETByIDHandler()`

## License

[Your License Here]

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

For issues related to:
- This MCP server: Open an issue in this repository
- Audiobookshelf: Visit [audiobookshelf.org](https://www.audiobookshelf.org/)
- MCP Protocol: See [modelcontextprotocol.io](https://modelcontextprotocol.io/)
