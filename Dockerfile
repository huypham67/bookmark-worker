# Stage 1: BASE - Install Dependencies and Prepare Source
FROM golang:1.26-alpine AS base

RUN apk add --no-cache build-base git

WORKDIR /opt/app

# Separate copy go.mod/sum for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Stage 2: BUILD - Compile Binary
FROM base AS build

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -o bookmark-worker \
    ./cmd/worker/main.go

# Stage 3: TEST-EXEC - Run Tests and Generate Coverage Reports
FROM base AS test-exec

ARG COVERAGE_EXCLUDE
ARG COVERPKG=./...
ENV _OUTPUTDIR=/tmp/coverage

RUN mkdir -p ${_OUTPUTDIR}

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go test ./... \
      -coverprofile=coverage.tmp \
      -covermode=atomic \
      -coverpkg=${COVERPKG} \
      -p 1 \
      -timeout 60s && \
    grep -v -E "${COVERAGE_EXCLUDE}" coverage.tmp > ${_OUTPUTDIR}/coverage.out && \
    go tool cover -html=${_OUTPUTDIR}/coverage.out -o ${_OUTPUTDIR}/coverage.html

# Stage 4: TEST-REPORT - Extract Coverage Reports for CI/CD
FROM scratch AS test

COPY --from=test-exec /tmp/coverage/coverage.out /coverage.out
COPY --from=test-exec /tmp/coverage/coverage.html /coverage.html

# Stage 5: FINAL - Production Image
FROM alpine:3.19 AS final

ENV SERVICE_USER=svc-worker \
    SERVICE_GROUP=svc-worker \
    TZ=Asia/Ho_Chi_Minh

RUN addgroup -S ${SERVICE_GROUP} && \
    adduser -S ${SERVICE_USER} -G ${SERVICE_GROUP}

WORKDIR /app

COPY --from=build /opt/app/bookmark-worker .

RUN chown -R ${SERVICE_USER}:${SERVICE_GROUP} /app && \
    apk add --no-cache tzdata && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone

USER ${SERVICE_USER}

# A worker has no HTTP port; liveness is the binary process itself.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep -x bookmark-worker || exit 1

CMD ["./bookmark-worker"]
