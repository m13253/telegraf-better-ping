package main

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/m13253/telegraf-better-ping/params"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (app *appState) startSenders() {
	var wg sync.WaitGroup
	for i := range app.Destinations {
		dest := &app.Destinations[i]
		wg.Add(1)
		go app.startSender(dest, &wg)
	}
	wg.Wait()
	os.Exit(1)
}

func (app *appState) startSender(dest *destinationState, wg *sync.WaitGroup) {
	ipv4Conn, ipv6Conn, err := app.createSendConn(dest.Params)
	if err != nil {
		log.Println(err)
		wg.Done()
		return
	}
	delay := time.Duration(rand.Int63n(int64(dest.Params.Interval)))
	fmt.Printf("# PING %s with %d bytes of data, will start in %.3f seconds.\n", strings.ReplaceAll(dest.Params.Destination, "\n", "\n# "), dest.Params.Size, delay.Seconds())
	time.Sleep(delay)
	var (
		seq   uint16
		crypt cipher.AEAD
	)

	ticker := time.NewTicker(dest.Params.Interval)
	defer ticker.Stop()
out:
	for ; ; <-ticker.C {
		if seq == 0 {
			if crypt != nil {
				dest.Crypt[1].Store(crypt)
			}
			crypt, err = app.RandomGenerator.Chacha20Poly1305()
			if err != nil {
				log.Fatalf("failed to initialize destination %s: %v\n", dest.Params.Destination, err)
			}
			dest.Crypt[0].Store(crypt)
		}
		// Yes, integer overflow is defined behavior in Go.
		// https://go.dev/ref/spec#Integer_overflow
		seq++

		addrs, err := net.LookupHost(dest.Params.Destination)
		if err != nil {
			log.Printf("failed to lookup %s: %v\n", dest.Params.Destination, err)
			continue
		}
		var firstErr error
		for _, addr := range addrs {
			ipv4Packet, ipv6Packet := app.prepareRequestBody(dest, seq, crypt)
			if ipv6Conn != nil {
				if ipv6Addr, err := net.ResolveIPAddr("ip6", addr); err == nil {
					_, err = ipv6Conn.WriteTo(ipv6Packet, nil, ipv6Addr)
					if err == nil {
						continue out
					}
					firstErr = err
				}
			}
			if ipv4Conn != nil {
				if ipv4Addr, err := net.ResolveIPAddr("ip4", addr); err == nil {
					_, err = ipv4Conn.WriteTo(ipv4Packet, nil, ipv4Addr)
					if err == nil {
						continue out
					}
					firstErr = err
				}
			}
		}
		if firstErr != nil {
			log.Printf("failed to ping %s: %v\n", dest.Params.Destination, firstErr)
		} else {
			log.Printf("failed to ping %s: no available address\n", dest.Params.Destination)
		}
	}
}

func (app *appState) prepareRequestBody(dest *destinationState, seq uint16, crypt cipher.AEAD) (ipv4Packet, ipv6Packet []byte) {
	sendTime := time.Now()
	sendTimeSinceEpoch := sendTime.Sub(app.Epoch)
	unixTimeSec := sendTime.Unix()
	unixTimeMSec := sendTime.Nanosecond() / 1000

	var nonce [chacha20poly1305.NonceSize]byte
	binary.BigEndian.PutUint16(nonce[:2], dest.ID)
	binary.BigEndian.PutUint16(nonce[2:4], seq)
	binary.LittleEndian.PutUint64(nonce[4:12], uint64(unixTimeSec))

	var additional [8]byte
	binary.LittleEndian.PutUint64(additional[:], uint64(unixTimeMSec))

	payload := make([]byte, dest.Params.Size-32, dest.Params.Size-16)
	binary.BigEndian.PutUint64(payload[:8], uint64(sendTimeSinceEpoch))

	ciphertext := crypt.Seal(payload[:0], nonce[:], payload, additional[:])
	data := make([]byte, dest.Params.Size)
	copy(data[:8], nonce[4:12])
	copy(data[8:16], additional[:])
	copy(data[16:dest.Params.Size], ciphertext[:dest.Params.Size-16])

	body := icmp.Echo{
		ID:   int(dest.ID),
		Seq:  int(seq),
		Data: data,
	}
	ipv4Packet, err := (&icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Body: &body,
	}).Marshal(nil)
	if err != nil {
		panic(err)
	}
	ipv6Packet, err = (&icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest,
		Body: &body,
	}).Marshal(nil)
	if err != nil {
		panic(err)
	}
	return
}

func (app *appState) createSendConn(dest *params.DestinationParams) (ipv4Conn *ipv4.PacketConn, ipv6Conn *ipv6.PacketConn, err error) {
	switch dest.Protocol {
	case "ip":
		icmpConn, ipv4Err := icmp.ListenPacket("ip4:1", dest.Source)
		if ipv4Err == nil {
			ipv4Conn = icmpConn.IPv4PacketConn()
		}
		icmpv6Conn, ipv6Err := icmp.ListenPacket("ip6:58", dest.Source)
		if ipv6Err == nil {
			ipv6Conn = icmpv6Conn.IPv6PacketConn()
		}
		if ipv4Err != nil && ipv6Err != nil {
			err = fmt.Errorf("failed to create socket for destination %s: %w", dest.Destination, ipv4Err)
		}
	case "ip4":
		icmpConn, ipv4Err := icmp.ListenPacket("ip4:1", dest.Source)
		if ipv4Err == nil {
			ipv4Conn = icmpConn.IPv4PacketConn()
		}
		err = ipv4Err
	case "ip6":
		icmpv6Conn, ipv6Err := icmp.ListenPacket("ip6:58", dest.Source)
		if ipv6Err == nil {
			ipv6Conn = icmpv6Conn.IPv6PacketConn()
		}
		err = ipv6Err
	default:
		panic(fmt.Sprintf("unknown protocol: %q", dest.Protocol))
	}
	return
}
