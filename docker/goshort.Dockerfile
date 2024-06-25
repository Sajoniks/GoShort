FROM golang:1.22-alpine3.20 as build

RUN apk add --update alpine-sdk

WORKDIR src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY internal internal
COPY cmd/go-short cmd

RUN CGO_ENABLED=1 GOOS=linux go build -o /usr/local/bin/app ./cmd

FROM alpine:3.20
COPY --from=build /usr/local/bin/app /usr/local/bin/app

ENTRYPOINT ["app"]

