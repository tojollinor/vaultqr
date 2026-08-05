# VaultQR

VaultQR **v1.0.1** generates QR codes from URL parameters and redirects to a neutral URL afterwards.

## Behavior

- `content=...` shows only the QR code.
- `text=...` shows the QR code, clear text and a copy button.
- The browser is redirected to `/view`.
- The QR image is served through `/image`.
- Sessions exist only in memory and expire automatically.
- Optional secret protection returns HTTP `403` for missing or incorrect secrets.

## Environment variables

| Variable | Default | Description |
|---|---:|---|
| `PORT` | `8080` | Internal HTTP port |
| `SIZE` | `400` | Default QR size |
| `MAX_SIZE` | `1024` | Maximum allowed QR size |
| `SESSION_TTL_MINUTES` | `30` | Session lifetime |
| `SECRET` | empty | Optional URL secret |
| `LOG_LEVEL` | `normal` | `off`, `normal` or `debug` |

## Logging

### `LOG_LEVEL=off`

Logs startup information and errors only.

### `LOG_LEVEL=normal`

Additionally logs requests, status codes, duration, selected mode, QR size and payload length. The payload itself is not logged.

### `LOG_LEVEL=debug`

Additionally logs:

- whether `content` or `text` was used,
- the QR payload,
- selected non-sensitive request metadata.

Payload logging is limited to 500 characters. VaultQR never logs the configured secret, cookies, authorization headers or complete query strings.

> Debug logging may expose sensitive QR payloads. Enable it only temporarily.

## Examples

QR only:

```text
https://vault.example.de/qr/?content=Hello
```

QR and visible text:

```text
https://vault.example.de/qr/?text=Hello
```

With size and secret:

```text
https://vault.example.de/qr/?text=Hello&size=600&secret=YOUR_SECRET
```

## Docker Compose

```yaml
services:
  vaultqr:
    image: ghcr.io/tojollinor/vaultqr:latest
    container_name: vaultqr
    restart: unless-stopped
    environment:
      PORT: "8080"
      SIZE: "400"
      MAX_SIZE: "1024"
      SESSION_TTL_MINUTES: "30"
      SECRET: ""
      LOG_LEVEL: "normal"
    ports:
      - "8010:8080"
```

## Reverse proxy under `/qr`

The proxy must strip `/qr` before forwarding the request.

```caddy
handle_path /qr/* {
    reverse_proxy 10.1.1.7:8010
}
```

## Release 1.0.1

After pushing the files, create and push the tag:

```bash
git tag v1.0.1
git push origin v1.0.1
```

This publishes `ghcr.io/tojollinor/vaultqr:v1.0.1` through the included GitHub workflow.

## License

MIT
