
FROM golang:1.25-trixie as builder

COPY go.mod /build/
COPY go.sum /build/
COPY ali_sls.go /build/
RUN cd /build/ && go build -buildmode=c-shared -o ali_sls.so .

FROM fluent/fluent-bit:4.2.2 as fluent-bit
USER root

COPY --from=builder /build/ali_sls.so /fluent-bit/bin/
COPY --from=builder /build/ali_sls.h /fluent-bit/bin/

WORKDIR /fluent-bit

CMD ["./bin/fluent-bit", "-c", "etc/fluent-bit.conf", "-e", "bin/ali_sls.so"]