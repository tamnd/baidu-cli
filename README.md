# baidu-cli

A command line for Baidu. One pure-Go binary, no API key.

```bash
baidu hot                          # realtime hot search board (top 30)
baidu hot --tab movie -n 10        # top 10 movie searches
baidu suggest --query golang       # 10 autocomplete suggestions
baidu hot -o json                  # JSON output
baidu hot | jq                     # JSONL piped to jq
```

Output is a table at the terminal and JSONL when piped.

## Install

### Homebrew

```bash
brew install tamnd/tap/baidu
```

### Pre-built binaries

Download from [Releases](https://github.com/tamnd/baidu-cli/releases).

### Go

```bash
go install github.com/tamnd/baidu-cli/cmd/baidu@latest
```

### Docker

```bash
docker run --rm ghcr.io/tamnd/baidu:latest hot
```

## Commands

| Command | Description |
|---|---|
| `hot` | Baidu realtime or category hot search board |
| `suggest` | Baidu search suggestions |
| `version` | Print version information |

### `baidu hot`

```
Flags:
  --tab string   board tab: realtime|novel|movie|teleplay|car (default "realtime")
  -n int         limit number of records (default 30)
  -o string      output: table|json|jsonl|csv|tsv|url|raw
```

### `baidu suggest`

```
Flags:
  -Q, --query string   search term to get suggestions for (required)
  -o string            output: table|json|jsonl|csv|tsv|url|raw
```

## License

Apache-2.0. `baidu-cli` is an independent tool and is not affiliated with Baidu, Inc.
