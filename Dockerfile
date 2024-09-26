# Build Stage
FROM golang:1.22-alpine AS build-stage

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

ADD . /go/src/github.com/fishman/wikidata-processor
WORKDIR /go/src/github.com/fishman/wikidata-processor

RUN apk add --no-cache \
    build-base \
    git

RUN make build-alpine

# Final Stage
FROM alpine:latest

ARG GIT_COMMIT
ARG VERSION
LABEL REPO="https://github.com/fishman/wikidata-processor"
LABEL GIT_COMMIT=$GIT_COMMIT
LABEL VERSION=$VERSION

RUN apk add --no-cache --update \
    dumb-init

# Because of https://github.com/docker/docker/issues/14914
ENV PATH=$PATH:/opt/wikidata-processor/bin

WORKDIR /opt/wikidata-processor/bin

COPY --from=build-stage /go/src/github.com/fishman/wikidata-processor/bin/wikidata-processor /opt/wikidata-processor/bin/
RUN chmod +x /opt/wikidata-processor/bin/wikidata-processor

# Create appuser
RUN adduser -D -g '' wikidata-processor
USER wikidata-processor

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["/opt/wikidata-processor/bin/wikidata-processor"]
