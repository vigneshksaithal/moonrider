FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/simulator ./cmd/simulator

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 moonrider
COPY --from=build /out/ /usr/local/bin/
USER moonrider
EXPOSE 8080
ENTRYPOINT ["api"]
