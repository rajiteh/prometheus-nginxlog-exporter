FROM golang:1.9 as builder
COPY . /go/src/github.com/rajiteh/prometheus-nginxlog-exporter
WORKDIR /go/src/github.com/rajiteh/prometheus-nginxlog-exporter
RUN go get . && CGO_ENABLED=0 go build -a -installsuffix cgo . && chmod +x prometheus-nginxlog-exporter

FROM scratch
COPY --from=builder /go/src/github.com/rajiteh/prometheus-nginxlog-exporter/prometheus-nginxlog-exporter .
ENTRYPOINT ["/prometheus-nginxlog-exporter"]
