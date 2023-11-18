# telegraf-better-ping

A better Ping monitoring plugin for Telegraf / InfluxDB

## Why the stock Ping plugin is not good enough?

<https://github.com/influxdata/telegraf/issues/11145#issuecomment-1809246992>

## Command line interface

This program can be run independently without Telegraf.
```
Usage:
  telegraf-better-ping {[OPTIONS] [--dest=]DESTINATION} [[OPTIONS] [--dest=]DESTINATION]...

Options:
  --comment=COMMENT     Comment of the following destination.
  [--dest=]DESTINATION  The destination address to send packets to.
                        The text "--dest=" can be omitted.
  --host-tag TAG        Add an extra "host" tag to the InfluxDB entries.
  --prefer-ipv6         Prefer IPv6 / ICMPv6 protocol,
                        fallback to IPv4 / ICMP. The default mode.
  -4                    Use IPv4 / ICMP protocol.
  -6                    Use IPv6 / ICMPv6 protocol.
  -I SOURCE             The source address to send packets from.
  -i INTERVAL           Wait INTERVAL seconds between sending each packet.
                        Must be greater or equal to 0.002 seconds.
  -s SIZE               The number of data bytes to be sent. The default is 56.
                        Must be between 40 and 65528.

Notes:
  All options, except for --comment, only affect the destinations followed by.
  The option --comment only affects the single destination followed by.
  The last command line argument must be a destination.
```

For example:
```bash
sudo ./telegraf-better-ping \
    --comment='Cloudflare DNS IPv4 (main)'   1.1.1.1 \
    --comment='Cloudflare DNS IPv4 (backup)' 1.0.0.1 \
    --comment='Cloudflare DNS IPv6 (main)'   2606:4700:4700::1111 \
    --comment='Cloudflare DNS IPv6 (backup)' 2606:4700:4700::1001 \
    --comment='Google DNS IPv4 (main)'   8.8.8.8 \
    --comment='Google DNS IPv4 (backup)' 8.8.4.4 \
    --comment='Google DNS IPv6 (main)'   2001:4860:4860::8888 \
    --comment='Google DNS IPv6 (backup)' 2001:4860:4860::8844 \
    --comment='Google WWW IPv4'     -4 www.google.com \
    --comment='Google WWW IPv6'     -6 www.google.com \
    --comment='Cloudflare WWW IPv4' -4 www.cloudflare.com \
    --comment='Cloudflare WWW IPv6' -6 www.cloudflare.com
```

It prints out PING responses to standard output, in the [InfluxDB line protocol](https://docs.influxdata.com/influxdb/v2/reference/syntax/line-protocol/) format.
```
# PING 192.168.0.2 with 56 bytes of data, will start in 0.250 seconds.
# PING 2001:db8::2 with 56 bytes of data, will start in 0.750 seconds.
ping,dest=192.168.0.2 size=64u,reply_from="192.168.0.2",reply_to="192.168.0.1",icmp_id=43690u,icmp_seq=1u,hop_limit=64u,rtt=0.001000000 1700000000250000000
ping,dest=2001:db8::2 size=64u,reply_from="2001:db8::2",reply_to="2001:db8::1",icmp_id=52428u,icmp_seq=1u,hop_limit=64u,rtt=0.001000000 1700000000750000000
ping,dest=192.168.0.2 size=64u,reply_from="192.168.0.2",reply_to="192.168.0.1",icmp_id=43690u,icmp_seq=2u,hop_limit=64u,rtt=0.001000000 1700000001250000000
ping,dest=2001:db8::2 size=64u,reply_from="2001:db8::2",reply_to="2001:db8::1",icmp_id=52428u,icmp_seq=2u,hop_limit=64u,rtt=0.001000000 1700000001750000000
ping,dest=192.168.0.2 size=64u,reply_from="192.168.0.2",reply_to="192.168.0.1",icmp_id=43690u,icmp_seq=3u,hop_limit=64u,rtt=0.001000000 1700000002250000000
ping,dest=2001:db8::2 size=64u,reply_from="2001:db8::2",reply_to="2001:db8::1",icmp_id=52428u,icmp_seq=3u,hop_limit=64u,rtt=0.001000000 1700000002750000000
ping,dest=192.168.0.2 size=64u,reply_from="192.168.0.2",reply_to="192.168.0.1",icmp_id=43690u,icmp_seq=4u,hop_limit=64u,rtt=0.001000000 1700000003250000000
ping,dest=2001:db8::2 size=64u,reply_from="2001:db8::2",reply_to="2001:db8::1",icmp_id=52428u,icmp_seq=4u,hop_limit=64u,rtt=0.001000000 1700000003750000000
# ...
```

## Running in Docker

### 1. Setting up database storage

First, create a directory outside Docker to store databases, so you will not lose it during future upgrades:
```bash
$ mkdir -p /var/lib/docker-volumes/{grafana,influxdb}
```

### 2. Setting up InfluxDB

```bash
$ docker pull influxdb:latest
$ docker create --name influxdb-1 \
    -p 127.0.0.1:8086:8086/tcp \
    -v /var/lib/docker-volumes/influxdb:/var/lib/influxdb2 \
    influxdb:latest
$ docker start influxdb-1
```

Open your browser, go to `http://127.0.0.1:8086` to go through the first-time setup.

Take notes of:
* Your operator token
* Your username and password
* Your organization name and bucket name

Log into `http://127.0.0.1:8086`, choose “Load Data” → “Bucket” from the left-side menu.

Choose “Settings” next to your bucket, select a retention period as your wish. Any data older than the specified period will be deleted.

### 3. Setting up Telegraf-better-ping

#### 3.a. Easy method: Passing configuration through environment variables.

Log into `http://127.0.0.1:8086`, choose “Load Data” → “API Tokens” from the left-side menu.

Choose “Generate API Token” → “Custom API Token”. Use the following settings:
* Description: `Telegraf-better-ping`
* Buckets → `<your bucket name>`: Write

Take note of your Telegraf-better-ping token.

```bash
$ docker pull m13253/telegraf-better-ping:latest
$ docker create --name telegraf-better-ping-1 \
    --link influxdb-1:influxdb \
    -e INFLUX_URL='http://influxdb:8086' \
    -e INFLUX_TOKEN='<your Telegraf-better-ping token>' \
    -e INFLUX_ORG='<your organization name>' \
    -e INFLUX_BUCKET='<your bucket name>' \
    -e TELEGRAF_BETTER_PING_ARGS='<your telegraf-better-ping command line arguments>'
    m13253/telegraf-better-ping:latest
$ docker start telegraf-better-ping-1
```
Refer to the Section [Command line interface](#command-line-interface) to learn how to configure `TELEGRAF_BETTER_PING_ARGS`.

#### 3.b. Alternative method: Use InfluxDB to distribute Telegraf configuration files.

Log into `http://127.0.0.1:8086`, choose “Load Data” → “Telegraf” from the left-side menu.

Choose “Create a Telegraf Configuration”. Use the following settings:
* Bucket: `<your bucket name>`
* Source: Execd

Make the following modification:
```toml
[[inputs.execd]]
  command = [
    "telegraf-better-ping",
    "<my", "telegraf-better-ping", "command", "line", "arguments", "but", "splitted>",
  ]
```
Refer to the Section [Command line interface](#command-line-interface) to learn how to configure the command line arguments.

Click “Save and Test”. Take notes of:
* The API token
* The configuration URL, but change `127.0.0.1` to `telegraf`

Edit the newly added Telegraf configuration, make the following modifications:
```toml
[agent]
  interval = "1s"
  flush_interval = "1s"
  precision = "1ns"
[[outputs.influxdb_v2]]
  urls = ["http://telegraf:8086"]
```

```bash
$ docker pull m13253/telegraf-better-ping:latest
$ docker create --name telegraf-better-ping-1 \
    --link influxdb-1:influxdb \
    -e INFLUX_TOKEN='<your API token>' \
    m13253/telegraf-better-ping:latest \
    --config 'http://telegraf:8086/api/v2/telegrafs/<my configuration URL>'
$ docker start telegraf-better-ping-1
```

### 4. Setting up Grafana

Log into `http://127.0.0.1:8086` again, choose “Load Data” → “API Tokens” from the left-side menu.

Choose “Generate API Token” → “Custom API Token”. Use the following settings:
* Description: `Grafana`
* Buckets → `<your bucket name>`: Read

Take note of your Grafana token.

```bash
$ docker pull grafana/grafana:latest
$ docker create --name grafana-1 \
    --link influxdb-1:influxdb \
    -p 127.0.0.1:3000:3000/tcp \
    -v /var/lib/docker-volumes/grafana:/var/lib/grafana \
    grafana/grafana:latest
$ docker start grafana-1
```

Go to `http://127.0.0.1:3000`, log in with username `admin` and password `admin`. Then change your password. You can also change your username in your profile settings.

Choose “Connections” → “Data sources” from the left-side menu.

Add a new data source using the following settings:
* Type: InfluxDB
* Query language: Flux
* URL: `http://influxdb:8086`
* Basic auth: off
* Organization: `<your organization name>`
* Token: `<your Grafana token>`
* Min time interval: 1s

### 5. Designing your Grafana dashboard

Go to `http://127.0.0.1:3000`, choose “Dashboards” from the left-side menu.

Choose the “⚙” icon in the top-right corner. Use the following settings:
* Title: `Ping`
* Refresh live dashboards: on

Choose “Variables”, add a new variable. Use the following settings:
* Name: dest
* Show on dashboard: Nothing
* Data source: InfluxDB
* Query:
  ```go
  import "influxdata/influxdb/v1"
  v1.tagValues(
      bucket: "<your bucket name>",
      tag: "dest",
      predicate: (r) => r._measurement == "ping",
      start: v.timeRangeStart,
      stop: v.timeRangeStop
  )
  ```
* Multi-value: yes
* Include All option: yes

(**Note:** Alternatively, you may want to use `"comment"` instead of `"dest"` to distinguish PING destinations.)

Choose “Run query”, make sure it shows all your PING destinations. Then, choose “Apply”.

Choose “Close” in the top-right corner.

Choose “Add” → “Visualization” in the top-right corner. Use the following settings:
* Query:
  * Data source: InfluxDB
  * Query:
    ```go
    from(bucket: "<your bucket name>")
    |> range(start: v.timeRangeStart, stop:v.timeRangeStop)
    |> filter(fn: (r) =>
      r._measurement == "ping" and
      r._field == "rtt" and
      r.dest == "${dest}"
    )
    |> aggregateWindow(every: v.windowPeriod, fn: max, createEmpty: false)
    ```
    (**Note:** Alternatively, you may want to use `"avg"` instead of `"max"` if you care about the average round-trip-time within aggregation windows.)
* Panel options:
  * Title: `Ping: ${dest}`
  * Repeat options:
    * Repeat by variable: `dest`
    * Max per row: 4
* Legend:
  * Visibility: off
* Graph styles:
  * Line interpolation: Step after
  * Fill opacity: 50
  * Gradient mode: Scheme
* Standard options:
  * Unit: `seconds (s)`
  * Min: 0
  * Color scheme: Green-Yellow-Red (by value)

Choose “Apply” in the top-right corner.

Then, choose “💾” icon in the top-right corner. Save your dashboard.

## Caveats

### DNS Caching

Telegraf-better-ping does not cache DNS responses. Therefore, the provided Docker container image has [Dnsmasq](https://dnsmasq.org) preinstalled, which caches DNS responses for Telegraf-better-ping.

If you run Telegraf-better-ping without the provided Docker container image, you need to ensure DNS caching is working properly to prevent Telegraf-better-ping from sending out too one DNS request per interval.

### IPv6 connectivity

Docker comes with no IPv6 connectivity by default.

Please refer to the [Docker manuals](https://docs.docker.com/config/daemon/ipv6/) to enable IPv6 support.

Alternatively, you can also run Telegraf-better-ping [using the host network](https://docs.docker.com/network/network-tutorial-host/) without enabling IPv6 inside Docker networks. However, be aware that host network may not support `--link`.
