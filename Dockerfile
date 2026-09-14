# syntax=docker/dockerfile:1

# BUILDPLATFORM keeps the Go toolchain native; Go cross-compiles to the
# target platform, so multi-arch builds do not need QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-X github.com/Leechael/gemini-web-cli/cmd.Version=${VERSION} -X github.com/Leechael/gemini-web-cli/cmd.BuildTime=${BUILD_TIME}" \
      -o /out/gemini-web-cli .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gemini-web-cli /usr/local/bin/gemini-web-cli
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gemini-web-cli"]
CMD ["serve", "--host", "0.0.0.0", "--port", "8080"]
