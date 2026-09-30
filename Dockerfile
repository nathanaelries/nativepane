ARG GO_VERSION=1.26.2
FROM golang:${GO_VERSION} AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOTELEMETRY=off
COPY . .
RUN go test ./... && go vet ./... \
    && go run ./cmd/notices /notices.txt \
    && go build -trimpath -ldflags="-s -w" -o /nativepane . \
    && mkdir -p /empty-data && chmod 0700 /empty-data

FROM scratch
LABEL org.opencontainers.image.title="NativePane" \
      org.opencontainers.image.licenses="MIT AND BSD-3-Clause"
COPY --from=build /nativepane /nativepane
COPY --from=build /notices.txt /licenses/THIRD_PARTY_NOTICES.txt
COPY LICENSE LICENSING.md THIRD_PARTY_LICENSES.md /licenses/
COPY --from=build --chown=65532:65532 /empty-data /data
USER 65532:65532
ENV BIND_ADDR=0.0.0.0 PORT=8080 DATA_DIR=/data AUTH=none
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/nativepane"]
