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
ADD cmd/atarraya-webhook/main.go ./
ADD cmd/atarraya-webhook/webhook.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w'

FROM alpine:3.10
ENV TLS_CERT_FILE=/var/lib/secrets/cert.crt \
    TLS_KEY_FILE=/var/lib/secrets/cert.key
RUN apk --no-cache add bash
# Copy the CA certificates
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
EXPOSE 8443
# Copy and use the dummy user 
COPY --from=builder /etc_passwd /etc/passwd
COPY --from=builder src/atarraya /bin/atarraya
COPY ./build/package/entrypoint.sh /bin/entrypoint.sh
USER dummy
ENTRYPOINT ["entrypoint.sh"]
CMD []