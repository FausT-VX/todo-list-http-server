# Satage 1 - Build
FROM golang:1.23 AS build

WORKDIR /app
ENV APP_NAME="todo_server"

COPY web/ ./web/
COPY go.mod go.sum ./
COPY database/scheduler.db ./database/scheduler.db

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o todo_server .

# Satage 2 - Production
FROM alpine:edge

WORKDIR /app
ENV APP_NAME="todo_server"

COPY --from=build /app/todo_server .
COPY --from=build /app/web/ /app/web/
COPY --from=build /app/tests/settings.go /app/tests/settings.go
COPY --from=build /app/database/scheduler.db /app/database/scheduler.db

ENV TODO_PORT=7540
ENV TODO_DBFILE=./database/scheduler.db
ENV TODO_PASSWORD=123
EXPOSE $TODO_PORT

ENTRYPOINT ["/app/todo_server"]