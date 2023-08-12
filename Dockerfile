FROM golang:1.20 AS builder

WORKDIR /build
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY pkg/ pkg/
COPY cmd/ cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o netflow-collector cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /build/netflow-collector .

USER 65532:65532

ENTRYPOINT ["/netflow-collector"]

