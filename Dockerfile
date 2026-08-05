FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/vaultqr \
    ./cmd/vaultqr

FROM scratch

COPY --from=build /out/vaultqr /vaultqr

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/vaultqr"]