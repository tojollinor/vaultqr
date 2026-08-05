# Changelog

## [1.0.1] - 2026-08-05

### Added

- `LOG_LEVEL` with the values `off`, `normal` and `debug`.
- Debug logging of whether `content` or `text` was used.
- Debug logging of the QR payload, limited to 500 characters.
- Startup log now includes the application version and active log level.

### Security

- Secrets, cookies and authorization headers are never logged.
- Query strings are not included in request logs.

## [1.0.0] - 2026-08-05

- Initial release.
