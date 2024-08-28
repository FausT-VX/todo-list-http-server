# Satage 1 - Build
FROM golang:1.23 AS build

WORKDIR /app
ENV APP_NAME="todo_server"

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o todo_server .

# Satage 2 - Production
FROM alpine:edge

WORKDIR /app
ENV APP_NAME="todo_server"

COPY --from=build /app/todo_server .
COPY web/ ./web/
COPY scheduler.db scheduler.db

ENV TODO_PORT=7540
ENV TODO_DBFILE=../scheduler.db
ENV TODO_PASSWORD=123
EXPOSE $TODO_PORT

ENTRYPOINT ["/app/todo_server"]