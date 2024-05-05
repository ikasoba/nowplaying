FROM golang:1.22.2 AS build

WORKDIR /app

COPY go.mod go.sum main.go ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go build -o nowplaying

FROM golang:1.22.2

WORKDIR /app

COPY views views
COPY --from=build /app/nowplaying .

CMD ["./nowplaying"]