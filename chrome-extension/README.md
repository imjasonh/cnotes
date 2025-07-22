# GitHub Git Notes Viewer

A Chrome extension that displays git notes on GitHub commit pages via a popup interface. Click the extension icon while viewing a GitHub commit to see all available git notes.

## Features

- 🔍 Works on any GitHub commit page - just click the extension icon
- 🔢 Shows badge with note count on commits that have notes
- 📝 Displays all available git notes refs for a commit
- 🏷️ Supports multiple notes refs with tabbed interface
- 🎨 Clean popup UI with GitHub-style design
- ⚡ Caches results to minimize API calls
- 🚫 No authentication required (public repos only)

## Installation

### From Source (Development)

1. Clone this repository:
   ```bash
   git clone https://github.com/imjasonh/chrome-github-notes.git
   cd chrome-github-notes
   ```

2. Generate icon files from the SVG (optional):
   ```bash
   # Using ImageMagick or similar tool
   convert icons/icon.svg -resize 16x16 icons/icon-16.png
   convert icons/icon.svg -resize 48x48 icons/icon-48.png
   convert icons/icon.svg -resize 128x128 icons/icon-128.png
   ```

3. Load the extension in Chrome:
   - Open Chrome and navigate to `chrome://extensions`
   - Enable "Developer mode" in the top right
   - Click "Load unpacked"
   - Select the `chrome-github-notes` directory

## Usage

1. Navigate to any GitHub commit page (e.g., `https://github.com/owner/repo/commit/abc123`)
2. If the commit has notes, you'll see a blue badge on the extension icon showing the count
3. Click the extension icon to view the notes
4. Click on tabs to switch between different notes refs if multiple exist

The extension automatically checks for notes when you visit GitHub commit pages and displays a badge indicator when notes are found.

## How It Works

The extension uses the GitHub API to fetch git notes:

1. Detects when you're on a GitHub commit page
2. Extracts the repository and commit information from the URL
3. Makes API calls to fetch available notes refs
4. For each ref, fetches the notes content for the specific commit
5. Displays the notes in a formatted view

## API Rate Limits

- **Unauthenticated**: 60 requests per hour per IP address
- The extension caches results for 5 minutes to minimize API usage
- Rate limit errors are displayed clearly to the user

## Supported Note Formats

The extension recognizes and formats several note types:

### cnotes Format
If notes were created by [cnotes](https://github.com/imjasonh/cnotes), they're displayed with:
- Session information
- Claude version
- Tools used
- Formatted conversation excerpts

### JSON Format
JSON notes are pretty-printed with syntax highlighting

### Plain Text
Plain text notes are displayed in a monospace font

## Development

### Project Structure
```
chrome-github-notes/
├── manifest.json      # Extension manifest (V3)
├── background.js      # Service worker for API calls
├── popup.html        # Extension popup interface
├── popup.js          # Popup logic and UI handling
├── styles.css        # Popup styles
├── utils.js          # Utility functions (placeholder)
└── icons/            # Extension icons
```

### Key Components

- **popup.js**: Detects current tab URL and displays notes in popup
- **background.js**: Handles GitHub API calls to bypass CORS restrictions
- **styles.css**: Clean popup UI with GitHub-inspired design

## Limitations

- Only works with public repositories (no authentication)
- Limited to 60 API requests per hour
- Notes must be pushed to GitHub (`git push origin refs/notes/*`)
- Only displays notes on individual commit pages

## Future Enhancements

- [ ] OAuth authentication for private repos and higher rate limits
- [ ] Display notes on commit list pages
- [ ] User preferences and configuration
- [ ] Export functionality
- [ ] Support for more note formats
- [ ] Integration with git notes management tools

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE file for details

## Related Projects

- [cnotes](https://github.com/imjasonh/cnotes) - Git notes for Claude conversations
- [git-notes](https://git-scm.com/docs/git-notes) - Git notes documentation