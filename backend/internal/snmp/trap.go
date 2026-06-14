package snmp

import (
	"fmt"
	"net"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// Trap is a normalized SNMP trap/notification.
type Trap struct {
	Source    string            `json:"source"`
	Community string            `json:"community"`
	Bindings  map[string]string `json:"bindings"`
	At        time.Time         `json:"at"`
}

// TrapHandler consumes parsed traps.
type TrapHandler func(Trap)

// TrapListener receives SNMP traps on UDP (default :162) and dispatches them.
type TrapListener struct {
	Addr    string // e.g. "0.0.0.0:162"
	handler TrapHandler
	l       *g.TrapListener
}

// NewTrapListener builds a listener.
func NewTrapListener(addr string, h TrapHandler) *TrapListener {
	if addr == "" {
		addr = "0.0.0.0:162"
	}
	return &TrapListener{Addr: addr, handler: h}
}

// ListenAndServe blocks serving traps until Close is called.
func (t *TrapListener) ListenAndServe() error {
	t.l = g.NewTrapListener()
	t.l.OnNewTrap = func(p *g.SnmpPacket, addr *net.UDPAddr) {
		t.handler(parseTrap(p, addr))
	}
	return t.l.Listen(t.Addr)
}

// Close stops the listener.
func (t *TrapListener) Close() {
	if t.l != nil {
		t.l.Close()
	}
}

func parseTrap(p *g.SnmpPacket, addr *net.UDPAddr) Trap {
	tr := Trap{
		Community: p.Community,
		Bindings:  map[string]string{},
		At:        time.Now().UTC(),
	}
	if addr != nil {
		tr.Source = addr.IP.String()
	}
	for _, v := range p.Variables {
		tr.Bindings[v.Name] = fmt.Sprintf("%v", decodePDU(v))
	}
	return tr
}
