# Clipshare

Simple REST clipboard service and web/CLI clients.

The clipboard contents automatically clear after 60s.

## Security and access control

Clipshare does not provide any authentication mechanism. Deploy it behind
Tailscale (or similar) and delegate access control to it.

## Quick start

### Nix

Run the server:

```bash
nix run github:aldur/clipshare#server
# clipshare-server starting on http://localhost:8080

# Set the HOST and PORT environment variables to customize where 
# the server binds.
```

Now, from another terminal:

```bash
nix shell github:aldur/clipshare

echo "hello world" | clipshare set
```

And from another terminal:

```bash
nix run github:aldur/clipshare -- get
# "hello world"
```

If you wait 60s, the clipboard will automatically clear.

## Client usage

### Command line

See the CLI for help:

```bash
clipshare -h
```

Set `CLIPSHARE_URL` or use the `-u`/`--url` flag to point the client to your
`clipshare-server` instance.

### Web

Navigate to your `clipshare-server` instance (`http://localhost:8080` by
default) to find a simple HTTP client.

## REST API specs

See [openapi.yaml](./openapi.yaml).

## Deployment

The included [Nix flake](./flake.nix) provides:

1. A NixOS module for the _server_.
1. A Docker image for the _server_.
1. A home-manager module for the _client_.
1. Packages for both the _server_ and the _client_.

Being `go` code, you can also just build the server/client and copy them where
you need them.

## Development

### Nix

You can use `nix` to get a development shell (`nix develop`), build the project
`nix build` and test it `nix flake check`.
