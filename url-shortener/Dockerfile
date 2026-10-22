# Build stage: static binary without CGO
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /url-shortener .

# Runtime stage: distroless, non-root, ~10 MB
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /url-shortener /url-shortener
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/url-shortener"]
