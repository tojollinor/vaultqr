# VaultQR

VaultQR erzeugt QR-Codes aus URL-Parametern und leitet danach auf die neutrale Adresse `/view` weiter.

## Verhalten

- `content=...` zeigt nur den QR-Code.
- `text=...` zeigt QR-Code, Klartext und einen Kopieren-Button.
- Nach dem ersten Aufruf erfolgt eine Weiterleitung auf `/view`.
- Die Nutzdaten werden zeitlich begrenzt im Arbeitsspeicher gespeichert.
- Ein zufälliges Sitzungstoken liegt in einem HttpOnly-Cookie.
- Nach einem Container-Neustart sind bestehende Sitzungen nicht mehr verfügbar.

## Umgebungsvariablen

| Variable | Standard | Bedeutung |
|---|---:|---|
| `PORT` | `8080` | interner HTTP-Port |
| `SIZE` | `400` | Standardgröße ohne URL-Parameter |
| `MAX_SIZE` | `1024` | maximal erlaubte Größe |
| `SESSION_TTL_MINUTES` | `30` | Gültigkeit der neutralen Ansicht |
| `SECRET` | leer | optionaler Zugriffsschutz |

## Beispiele

Nur QR-Code:

```text
https://vault.example.de/qr/?content=Hallo
```

QR-Code mit Klartext und Kopieren-Button:

```text
https://vault.example.de/qr/?text=Hallo
```

Mit eigener Größe:

```text
https://vault.example.de/qr/?text=Hallo&size=600
```

Mit Secret:

```text
https://vault.example.de/qr/?text=Hallo&secret=DEIN_SECRET
```

Nach erfolgreicher Verarbeitung steht im Browser nur noch:

```text
https://vault.example.de/qr/view
```

## Sicherheitshinweis

Die Weiterleitung entfernt die Daten aus der sichtbaren Adresszeile. Der ursprüngliche Aufruf kann trotzdem im Browser-Verlauf oder in vorgelagerten Proxy-Logs auftauchen. VaultQR selbst protokolliert keine Query-Parameter.

## Reverse Proxy

Bei `handle_path /qr/*` wird `/qr` entfernt. Dadurch wird `/qr/view` intern zu `/view`.

```caddy
handle_path /qr/* {
    reverse_proxy 10.1.1.7:8010
}
```

## Docker Compose

```yaml
services:
  vaultqr:
    build:
      context: .
      dockerfile: Dockerfile
    image: vaultqr:latest
    container_name: vaultqr
    restart: unless-stopped
    environment:
      PORT: "8080"
      SIZE: "400"
      MAX_SIZE: "1024"
      SESSION_TTL_MINUTES: "30"
      SECRET: ""
    ports:
      - "8010:8080"
```

## Lizenz

MIT
