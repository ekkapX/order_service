FROM golang:1.24.0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/myapp/main.go
COPY wait-for-it.sh .
CMD ["./server"]
