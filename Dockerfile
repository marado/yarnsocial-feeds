# Build
FROM golang:alpine AS build

RUN apk add --no-cache -U build-base git make

RUN mkdir -p /src

WORKDIR /src

# Copy Makefile
COPY Makefile ./

# Copy go.mod and go.sum and install and cache dependencies
COPY go.mod .
COPY go.sum .

# Install deps
RUN go mod download

# Copy sources
COPY *.go ./

# Version/Commit (there there is no .git in Docker build context)
# NOTE: This is fairly low down in the Dockerfile instructions so
#       we don't break the Docker build cache just be changing
#       unrelated files that actually haven't changed but caused the
#       COMMIT value to change.
ARG VERSION="0.0.0"
ARG COMMIT="HEAD"

# Build server binary
RUN make VERSION=$VERSION COMMIT=$COMMIT

# Runtime
FROM alpine:latest

RUN apk --no-cache -U add curl ca-certificates tzdata

WORKDIR /
VOLUME /data

# force cgo resolver
ENV GODEBUG=netdns=cgo

COPY --from=build /src/feeds /feeds

HEALTHCHECK CMD curl -qsfSL http://127.0.0.1:8000/health || exit 1
ENTRYPOINT ["/feeds"]
CMD [""]
