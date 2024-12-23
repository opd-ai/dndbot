# Adventure Forge 🎲✨

An AI-powered D&D adventure generator that creates complete, playable RPG content using Claude AI. Generate professional-grade adventures in minutes, complete with illustrations, maps, and organized documentation.

## Core Features

- **Real-Time Adventure Generation**
  - Complete D&D adventure modules
  - Rich narrative content
  - System-agnostic design
  - Copyright-compliant material

- **Advanced Content Pipeline**
  - Table of contents generation
  - Cover pages and artwork
  - Dungeon design and mapping
  - Adventure content expansion
  - Illustration prompts
  - Content review and validation

- **Professional Output**
  - Structured markdown formatting
  - ZIP file packaging
  - PDF compilation
  - Organized file hierarchy

## Installation

1. **Prerequisites**
```bash
- Go 1.21.3 or higher
- Make
```

2. **Environment Setup**
```bash
# Clone the repository
git clone https://github.com/your-org/dndbot.git
cd dndbot

# Copy and edit configuration
cp config.mk.example config.mk
```

3. **Required Environment Variables**
```bash
export CLAUDE_API_KEY="your-api-key"
export HORDE_API_KEY="your-horde-key"  # Optional for image generation
export SD_WEBUI_URL="your-sd-url"      # Optional for local image generation
```

## Usage

### Running the Server
```bash
# Build and run with default settings
make run

# Run with custom arguments
make run args="-port 3000 -domain localhost"
```

### Docker Deployment
```bash
# Build Docker image
make docker

# Run container
make docker-run
```

## Configuration Options

Server configuration flags:
```bash
-paywall    Enable payment requirements
-tls        Enable TLS/HTTPS
-mail       Email for certificates
-domain     Server domain name
-port       Server port number
```

## API Documentation

See [API.md](API.md) for detailed API documentation.

## Development

### Project Structure
```
dndbot/
├── cmd/          # Command line tools
├── srv/          # Server implementation
│   ├── ui/       # Web interface
│   └── generator/# Core generation logic
├── src/          # Core library
└── static/       # Web assets
```

### Building from Source
```bash
# Format code
make fmt

# Build binary
make build

# Clean build artifacts
make clean
```

### Testing
```bash
# Run test suite
go test ./...

# Run with Firefox profile
make fox
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit changes with clear messages
4. Push to your branch
5. Create a Pull Request

## License

GNU Affero General Public License v3.0 - See [LICENSE.md](LICENSE.md)

## Security Considerations

- Rate limiting implemented
- CORS protection enabled
- Security headers configured
- TLS support available

## Technical Stack

- **Backend:** Go
- **API:** Claude AI (Anthropic)
- **Image Generation:** Stable Diffusion
- **Web Interface:** HTML/CSS/JavaScript
- **Storage:** File-based + In-memory cache

## Support

If you find this project useful, consider supporting the developer:

```
Monero Address: `43H3Uqnc9rfEsJjUXZYmam45MbtWmREFSANAWY5hijY4aht8cqYaT2BCNhfBhua5XwNdx9Tb6BEdt4tjUHJDwNW5H7mTiwe`
Bitcoin Address: `bc1qew5kx0srtp8c4hlpw8ax0gllhnpsnp9ylthpas`
```

## Acknowledgements

- Claude AI by Anthropic
- Stable Diffusion
- Go Chi router
- Contributors and maintainers
