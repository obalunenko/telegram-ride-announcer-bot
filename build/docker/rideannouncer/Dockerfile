FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24 AS builder

LABEL maintainer="oleg.balunenko@gmail.com"
LABEL org.opencontainers.image.source="https://github.com/obalunenko/scrum-report"
LABEL stage="base"

WORKDIR /src

COPY . .

ARG TARGETOS
ARG TARGETARCH
# The same build as `make compile-app`, without make and the repo scripts.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /app/rideannouncer ./cmd/rideannouncer


FROM alpine:3.24 AS final

ARG APK_CA_CERTIFICATES_VERSION=~20261020

# Install any runtime dependencies that are needed to run your application.
# Leverage a cache mount to /var/cache/apk/ to speed up subsequent builds.
RUN apk add --no-cache \
        "ca-certificates=${APK_CA_CERTIFICATES_VERSION}"

# Create a non-privileged user that the app will run under.
# See https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#user
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER ${UID}

# Copy the executable from the "build" stage.
COPY build/docker/rideannouncer/entrypoint.sh /
COPY --from=builder /app/ /

# Expose the port that the application listens on.
# EXPOSE 8080

# What the container should run when it is started.
ENTRYPOINT ["sh", "-c", "/entrypoint.sh"]
