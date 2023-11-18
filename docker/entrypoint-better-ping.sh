#!/bin/bash
set -e

install -Dm0644 /etc/resolv.conf /run/dnsmasq/resolv.conf
dnsmasq -i lo -a 127.0.0.53 -r /run/dnsmasq/resolv.conf
echo 'nameserver 127.0.0.53' >/etc/resolv.conf
exec /entrypoint.sh "$@"
