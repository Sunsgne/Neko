package probe

import (
	"context"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// attemptICMP sends one ICMP echo and measures RTT. It uses unprivileged ICMP
// (udp4 socket) which works on Linux when net.ipv4.ping_group_range permits;
// otherwise it returns an error and the sample counts as loss.
func attemptICMP(ctx context.Context, target string, timeout time.Duration) (time.Duration, error) {
	ipAddr, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return 0, err
	}
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("neko-probe")},
	}
	wb, err := msg.Marshal(nil)
	if err != nil {
		return 0, err
	}

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		_ = conn.SetDeadline(d)
	}

	start := time.Now()
	if _, err := conn.WriteTo(wb, &net.UDPAddr{IP: ipAddr.IP}); err != nil {
		return 0, err
	}

	rb := make([]byte, 1500)
	n, _, err := conn.ReadFrom(rb)
	if err != nil {
		return 0, err
	}
	rtt := time.Since(start)

	rm, err := icmp.ParseMessage(1, rb[:n]) // 1 = ICMPv4 protocol number
	if err != nil {
		return 0, err
	}
	if rm.Type != ipv4.ICMPTypeEchoReply {
		return 0, os.ErrDeadlineExceeded
	}
	return rtt, nil
}
