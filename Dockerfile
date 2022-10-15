ARG GOVERSION=1.19
FROM golang:${GOVERSION} AS build

WORKDIR /app
COPY . .
RUN make build

FROM scratch
COPY --from=build /app/out/bin/nodegraph-server /nodegraph-server
CMD ["/nodegraph-server"]

