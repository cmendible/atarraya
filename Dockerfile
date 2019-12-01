ARG GO_VERSION=1.13

FROM golang:${GO_VERSION}-alpine AS builder

# Get CA certificates and install git
RUN apk update \
    && apk add ca-certificates \
    && rm -rf /var/cache/apk/* \
    && update-ca-certificates \
    && apk add git
# Create a dummy user
RUN echo "dummy:x:1001:1001:Dummy:/:" > /etc_passwd
WORKDIR /src
ADD go.mod go.sum ./
RUN go mod download
ADD az-atarraya.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w'

FROM alpine:3.10
# Copy the CA certificates
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder src/az-atarraya /usr/local/bin/az-atarraya
# Copy and use the dummy user 
COPY --from=builder /etc_passwd /etc/passwd
USER dummy