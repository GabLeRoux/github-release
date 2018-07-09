FROM golang:alpine as builder

ARG user=itchio
ARG repo=gothub

RUN apk update && \
    apk upgrade && \
    apk add -u git && \
    go get github.com/$user/$repo

FROM alpine:latest
RUN apk --no-cache add ca-certificates jq

COPY --from=builder /go/bin/gothub /bin/gothub

CMD ["gothub"]
