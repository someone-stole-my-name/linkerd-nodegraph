FROM golang:1.19 AS build

WORKDIR /app
COPY . .
RUN make build

FROM scratch
COPY --from=build /app/out/bin/nodegraph-server /nodegraph-server
CMD ["/nodegraph-server"]

