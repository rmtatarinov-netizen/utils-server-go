FROM golang:1.22-alpine

WORKDIR /app

RUN apk add --no-cache gcc g++ make sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o /server ./cmd/server

EXPOSE 8000

ENV DB_PATH=/data/users.db \
    ACTIVE_USERS_LIMIT=5 \
    PORT=8000

VOLUME ["/data"]

CMD ["/server"]
