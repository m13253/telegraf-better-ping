package main

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/m13253/telegraf-better-ping/influxDB_escape"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type icmpResponse struct {
	Comment     string
	Destination string
	HasHopLimit bool
	HopLimit    uint8
	HostTag     string
	ID          uint16
	RecvTime    time.Time
	ReplyFrom   net.Addr
	ReplyTo     net.Addr
	RTT         time.Duration
	Seq         uint16
	Size        int
}

func (app *appState) startReceivers() {
	ipv4Conn, err := icmp.ListenPacket("ip4:1", "")
	if err != nil {
		log.Fatalf("failed to listen on ICMP protocol: %v\n", err)
	}
	ipv4PacketConn := ipv4Conn.IPv4PacketConn()
	ipv4PacketConn.SetControlMessage(ipv4.FlagTTL, true)
	ipv4PacketConn.SetControlMessage(ipv4.FlagDst, true)
	ipv6Conn, err := icmp.ListenPacket("ip6:58", "")
	if err != nil {
		log.Fatalf("failed to listen on ICMPv6 protocol: %v\n", err)
	}
	ipv6PacketConn := ipv6Conn.IPv6PacketConn()
	ipv6PacketConn.SetControlMessage(ipv6.FlagHopLimit, true)
	ipv6PacketConn.SetControlMessage(ipv6.FlagDst, true)
	go app.startIPv4Receiver(ipv4PacketConn)
	go app.startIPv6Receiver(ipv6PacketConn)
}

func (app *appState) printResponse(resp *icmpResponse) {
	rttInt := resp.RTT / 1000000000
	rttFrac := resp.RTT % 1000000000
	if rttFrac < 0 {
		rttFrac = -rttFrac
	}
	var sb strings.Builder
	sb.WriteString("ping,")
	if len(resp.HostTag) != 0 {
		sb.WriteString(fmt.Sprintf("host=%s,", influxDB_escape.EscapeKey(resp.HostTag)))
	}
	sb.WriteString(fmt.Sprintf("dest=%s", influxDB_escape.EscapeKey(resp.Destination)))
	if len(resp.Comment) != 0 {
		sb.WriteString(fmt.Sprintf(",comment=%s", influxDB_escape.EscapeKey(resp.Comment)))
	}
	sb.WriteString(fmt.Sprintf(" size=%du,reply_from=%s,", resp.Size, influxDB_escape.EscapeValue(resp.ReplyFrom.String())))
	if resp.ReplyTo != nil {
		sb.WriteString(fmt.Sprintf("reply_to=%s,", influxDB_escape.EscapeValue(resp.ReplyTo.String())))
	}
	sb.WriteString(fmt.Sprintf("icmp_id=%du,icmp_seq=%du,", resp.ID, resp.Seq))
	if resp.HasHopLimit {
		sb.WriteString(fmt.Sprintf("hop_limit=%du,", resp.HopLimit))
	}
	sb.WriteString(fmt.Sprintf("rtt=%d.%09d %d\n", rttInt, rttFrac, resp.RecvTime.UnixNano()))
	fmt.Print(sb.String())
}

func (app *appState) processResponse(size int, src, dst net.Addr, recvTime time.Time, hasHopLimit bool, hopLimit uint8, body *icmp.Echo) {
	if len(body.Data) < 40 {
		log.Printf("failed to decipher ICMP message from %s: body is less than 40 bytes long", src)
		return
	}

	var nonce [chacha20poly1305.NonceSize]byte
	binary.BigEndian.PutUint16(nonce[:2], uint16(body.ID))
	binary.BigEndian.PutUint16(nonce[2:4], uint16(body.Seq))
	copy(nonce[4:12], body.Data[:8])

	additional := body.Data[8:16]
	ciphertext := body.Data[16:]

	for i := range app.Destinations {
		dest := &app.Destinations[i]
		for j := 0; j < 2; j++ {
			if crypt, ok := dest.Crypt[j].Load().(cipher.AEAD); ok {
				payload, err := crypt.Open(nil, nonce[:], ciphertext, additional)
				if err != nil {
					continue
				}

				sendTimeSinceEpoch := time.Duration(binary.BigEndian.Uint64(payload[:8]))
				recvTimeSinceEpoch := recvTime.Sub(app.Epoch)
				rtt := recvTimeSinceEpoch - sendTimeSinceEpoch

				app.printResponse(&icmpResponse{
					Comment:     dest.Params.Comment,
					Destination: dest.Params.Destination,
					HasHopLimit: hasHopLimit,
					HopLimit:    hopLimit,
					HostTag:     dest.Params.HostTag,
					ID:          uint16(body.ID),
					RecvTime:    recvTime,
					ReplyFrom:   src,
					ReplyTo:     dst,
					RTT:         rtt,
					Seq:         uint16(body.Seq),
					Size:        size,
				})
			}
		}
	}
}

func (app *appState) startIPv4Receiver(ipv4Conn *ipv4.PacketConn) {
	var buf [65536]byte
	for {
		n, cm, src, err := ipv4Conn.ReadFrom(buf[:])
		if err != nil {
			log.Fatalf("failed to receive ICMP message: %v\n", err)
		}
		recvTime := app.IncreasingNow()
		var (
			hasTTL bool
			ttl    uint8
			dst    net.Addr
		)
		if cm != nil {
			hasTTL = true
			ttl = uint8(cm.TTL)
			dst = &net.IPAddr{IP: cm.Dst}
		}
		msg, err := icmp.ParseMessage(1, buf[:n])
		if err != nil {
			log.Printf("failed to decode ICMP message from %s: %v\n", src.String(), err)
			continue
		}
		if body, ok := msg.Body.(*icmp.Echo); ok {
			app.processResponse(n, src, dst, recvTime, hasTTL, ttl, body)
		}
	}
}

func (app *appState) startIPv6Receiver(ipv6Conn *ipv6.PacketConn) {
	var buf [65536]byte
	for {
		n, cm, src, err := ipv6Conn.ReadFrom(buf[:])
		if err != nil {
			log.Fatalf("failed to receive ICMPv6 message: %v\n", err)
		}
		recvTime := app.IncreasingNow()
		var (
			hasHopLimit bool
			hopLimit    uint8
			dst         net.Addr
		)
		if cm != nil {
			hasHopLimit = true
			hopLimit = uint8(cm.HopLimit)
			dst = &net.IPAddr{IP: cm.Dst}
		}
		msg, err := icmp.ParseMessage(58, buf[:n])
		if err != nil {
			log.Printf("failed to decode ICMPv6 message from %s: %v\n", src.String(), err)
			continue
		}
		if body, ok := msg.Body.(*icmp.Echo); ok {
			app.processResponse(n, src, dst, recvTime, hasHopLimit, hopLimit, body)
		}
	}
}
