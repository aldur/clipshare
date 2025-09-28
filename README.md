# Clipshare

Simple REST clipboard service and web/CLI clients.

The clipboard contents automatically clear after 60s.

## Security and access control

Clipshare does not provide any authentication mechanism. Deploy it behind
Tailscale (or similar) and delegate access control to it.

## Usage

```bash
# terminal 1: start clipshare
go run main.go

# terminal 2: set the clipboard
echo "hello world" | go run cmd/client/main.go set

# terminal 3: get the clipboard
go run cmd/client/main.go get
# wait 60s...
# clipboard clears automatically
```

## REST API specs

See [openapi.yaml](./openapi.yaml).

## Deployment

The included [Nix flake](./flake.nix) provides modules for NixOS and
home-manager.

## Development

### Nix

You can use `nix` to get a development shell (`nix develop`), build the project
`nix build` and test it `nix flake check`.

