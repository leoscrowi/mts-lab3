FROM golang:1.26.6-alpine
WORKDIR /app
COPY lab2/go.mod lab2/go.sum ./
RUN go mod download
COPY lab2/config ./config
COPY lab2/internal ./internal
COPY lab2/tests ./tests
CMD ["go", "test", "./tests", "-run", "^TestPostsChain$", "-count=1", "-v", "-timeout=60s"]
