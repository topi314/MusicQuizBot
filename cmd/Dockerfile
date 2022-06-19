FROM golang:1.18-alpine AS build

ARG VERSION=dev

WORKDIR /tmp/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-X 'main.version=${VERSION}'-w -s" -o music-quiz-bot cmd/main.go

FROM alpine

RUN apk add --no-cache libmpg1230-dev\
    ca-certificates \

COPY --from=build /tmp/app/music-quiz-bot /bin/music-quiz-bot

ENTRYPOINT ["/bin/music-quiz-bot"]
