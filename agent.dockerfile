FROM golang:1.21

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .


# agent
RUN CGO_ENABLED=0 GOOS=linux go build -o ./cmd/agent/agent ./cmd/agent

WORKDIR /app/cmd/agent
CMD ["./agent","-host", "server"]
#CMD ./agent
