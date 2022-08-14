FROM golang:latest AS build

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 \
  go build \
  -o /bin/nodegraph-server cmd/nodegraph-server/main.go

FROM scratch
COPY --from=build /bin/nodegraph-server /nodegraph-server
CMD ["/nodegraph-server"]

