FROM golang:1 as builder

ADD . /root/telegraf-better-ping
RUN go get -C /root/telegraf-better-ping -u -v && \
    go build -C /root/telegraf-better-ping -o /root/telegraf-better-ping/telegraf-better-ping -v

FROM telegraf

ENV INFLUX_URL=http://localhost:8086 \
    INFLUX_TOKEN= \
    INFLUX_ORG=organization \
    INFLUX_BUCKET=bucket \
    TELEGRAF_BETTER_PING_ARGS="localhost"
COPY --from=builder /root/telegraf-better-ping/telegraf.conf /etc/telegraf/
COPY --from=builder /root/telegraf-better-ping/telegraf-better-ping /usr/bin/
RUN setcap cap_net_raw,cap_net_bind_service+ep /usr/bin/telegraf-better-ping
