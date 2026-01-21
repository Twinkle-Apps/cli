# Twinkle CLI

A command-line interface for the Twinkle build API.

## Requirements

- [mise](https://mise.jdx.dev/) for tool management
- Go (managed via `mise`)

```sh
mise install
```

## Authentication

Set an API key in the environment or pass it via `--api-key` (the flag wins).

```sh
export TWINKLE_API_KEY=your-token
```

## Usage

```sh
twinkle --help
```

Check a build status:

```sh
twinkle build status <app-id> <build-id>
```

Wait for processing (max 300 seconds per call):

```sh
twinkle build wait <app-id> <build-id> --timeout 300
```

Upload a build archive (zip only):

```sh
twinkle build upload <app-id> ./MyApp.zip
```

Upload and wait for completion:

```sh
twinkle build upload <app-id> ./MyApp.zip --wait --timeout 300
```

Output JSON:

```sh
twinkle --json build status <app-id> <build-id>
```

## Configuration

- `TWINKLE_API_KEY`: API key used for authentication
- `TWINKLE_BASE_URL`: override API base URL (default: `https://app.usetwinkle.com`)

## Development

```sh
mise install
go test ./...
```

## License

MIT
