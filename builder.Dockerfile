FROM golang:1.9-alpine AS builder

ARG USER=gableroux
ARG REPO=github-release
ARG DEP_VERSION=v0.4.1

RUN apk update && \
  apk add \
  curl \
  git \
  make \
  zip

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/$DEP_VERSION/dep-linux-amd64 && chmod +x /usr/local/bin/dep

RUN mkdir -p /go/src/github.com/$USER/$REPO
WORKDIR /go/src/github.com/$USER/$REPO

COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

COPY . ./

RUN make

CMD ["/go/src/github.com/$USER/$REPO/github-release"]
