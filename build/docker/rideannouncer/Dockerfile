FROM --platform=$BUILDPLATFORM ghcr.io/obalunenko/go-tools:v1.6.10 AS builder

LABEL maintainer="oleg.balunenko@gmail.com"
LABEL org.opencontainers.image.source="https://github.com/obalunenko/scrum-report"
LABEL stage="base"

WORKDIR /src

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make compile-app
RUN mkdir -p /app && \
    cp ./bin/rideannouncer /app/


FROM alpine:3.21 AS final

ARG APK_CA_CERTIFICATES_VERSION=~20241121

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
USER appuser

# Copy the executable from the "build" stage.
COPY build/docker/rideannouncer/entrypoint.sh /
COPY --from=builder /app/ /

# Expose the port that the application listens on.
# EXPOSE 8080

# What the container should run when it is started.
ENTRYPOINT ["sh", "-c", "/entrypoint.sh"]
