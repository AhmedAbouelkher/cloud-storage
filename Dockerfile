FROM golang:1.18.4-alpine3.16

WORKDIR /usr/src/app

COPY .env.example ./.env
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./...

EXPOSE 8080

CMD ["app"]