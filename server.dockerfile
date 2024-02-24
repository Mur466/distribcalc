FROM golang:1.21

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .


# Daemon
RUN CGO_ENABLED=0 GOOS=linux go build -o ./cmd/server/server ./cmd/server

#optional?
#EXPOSE 8080

WORKDIR /app/cmd/server
CMD ["./server","-dbhost", "distribcalc.storage"]
