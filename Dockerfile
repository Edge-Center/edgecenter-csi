FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /build
ADD go.mod go.sum /build/
RUN go mod download -x
ADD cmd /build/cmd
ADD pkg /build/pkg
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ./ec-csi-plugin ./cmd/main.go

FROM alpine:3.18.4
LABEL name="ec-csi-plugin" \
      description="Edgecenter CSI Plugin" \
      distribution-scope="public" \
      summary="Edgecenter CSI Plugin" \
      help="none"
RUN apk add --no-cache rsync mount udev btrfs-progs e2fsprogs e2fsprogs-extra xfsprogs xfsprogs-extra ca-certificates curl blkid findmnt
COPY --from=builder /build/ec-csi-plugin /usr/local/bin/
ENTRYPOINT ["ec-csi-plugin"]