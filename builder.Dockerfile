FROM golang:alpine

RUN apk update && \
    apk upgrade && \
    apk add -u git make

RUN apk --no-cache add ca-certificates jq

WORKDIR /app
COPY . /app
RUN make

CMD ["/app/github-release"]
