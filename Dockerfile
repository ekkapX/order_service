FROM golang:1.25.1
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/app/main.go
COPY wait-for-it.sh .
CMD ["./server"]
