#!/bin/bash
set -e

install -Dm0644 /etc/resolv.conf /run/dnsmasq/resolv.conf
dnsmasq -i lo -a "$DNSMASQ_LISTEN_ADDR" -z -r /run/dnsmasq/resolv.conf -c 1000
echo "nameserver $DNSMASQ_LISTEN_ADDR" >/etc/resolv.conf
exec /entrypoint.sh "$@"
