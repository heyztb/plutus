# Step 1
# golang:alpine
FROM golang@sha256:3e935ab77ba5d71c7778054fbb60c029c1564b75266beeeb4223aa04265e16c1 AS builder

# Install git 
RUN apk update && apk add --no-cache git

ENV USER=appuser
ENV UID=10001

RUN adduser \
        --disabled-password \
        --gecos "" \
        --home "/nonexistent" \
        --shell "/sbin/nologin" \
        --no-create-home \
        --uid "${UID}" \
        "${USER}"

ENV GO111MODULE=on \
        CGO_ENABLED=0 \
        GOOS=linux \
        GOARCH=amd64

WORKDIR /build

COPY . .
RUN go mod download
RUN go mod verify
RUN go build -ldflags="-w -s" -o ./binary 

# Step 2

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /build/binary binary

USER appuser:appuser

ENTRYPOINT ["./binary"]
