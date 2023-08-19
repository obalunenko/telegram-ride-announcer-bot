FROM --platform=$BUILDPLATFORM golang:1.21-alpine3.18 AS builder

LABEL maintainer="oleg.balunenko@gmail.com"
LABEL org.opencontainers.image.source="https://github.com/obalunenko/scrum-report"
LABEL stage="base"


WORKDIR /src

ARG APK_BASH_VERSION=~5
ARG APK_GIT_VERSION=~2
ARG APK_MAKE_VERSION=~4
ARG APK_OPENSSH_VERSION=~9
ARG APK_GCC_VERSION=~12
ARG APK_BUILDBASE_VERSION=~0
ARG APK_BINUTILS_VERSION=~2

RUN apk add --no-cache \
    "bash=${APK_BASH_VERSION}" \
	"git=${APK_GIT_VERSION}" \
	"make=${APK_MAKE_VERSION}" \
	"openssh-client=${APK_OPENSSH_VERSION}" \
	"build-base=${APK_BUILDBASE_VERSION}" \
    "gcc=${APK_GCC_VERSION}" \
    "binutils-gold=${APK_BINUTILS_VERSION}"

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make compile-app
RUN mkdir -p /app && \
    cp ./bin/rideannouncer /app/


FROM alpine:3.18 AS final

ARG APK_CA_CERTIFICATES_VERSION=~20230506

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
