FROM golang:1.9-alpine AS builder

ARG USER=gableroux
ARG REPO=github-release

RUN apk update && \
    apk upgrade && \
    apk add -u git && \
    go get github.com/$USER/$REPO

FROM alpine:latest
RUN apk --no-cache add ca-certificates jq

COPY --from=builder /go/bin/github-release /bin/github-release

CMD ["github-release"]
