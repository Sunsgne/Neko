package snmp

import (
	"fmt"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// Standard OIDs polled by the engine.
const (
	OIDsysDescr    = "1.3.6.1.2.1.1.1.0"
	OIDsysObjectID = "1.3.6.1.2.1.1.2.0"
	OIDsysName     = "1.3.6.1.2.1.1.5.0"
	OIDifNumber    = "1.3.6.1.2.1.2.1.0"
	OIDifHCInBase  = "1.3.6.1.2.1.31.1.1.1.6"  // ifHCInOctets.<idx>
	OIDifHCOutBase = "1.3.6.1.2.1.31.1.1.1.10" // ifHCOutOctets.<idx>
	// MikroTik health (CPU/temp/mem) live under enterprise OIDs; CPU load:
	OIDmtxrCPUUsage = "1.3.6.1.2.1.25.3.3.1.2.1"
)

// Version selects the SNMP protocol version.
type Version string

const (
	V2c Version = "v2c"
	V3  Version = "v3"
)

// Credential carries SNMP auth material. For v2c only Community is used; for
// v3 the User/Auth/Priv fields apply (authPriv recommended).
type Credential struct {
	Version   Version
	Community string
	// v3
	User      string
	AuthProto string // SHA | MD5
	AuthPass  string
	PrivProto string // AES | DES
	PrivPass  string
}

// Engine performs SNMP GETs against a target.
type Engine struct {
	Port    uint16
	Timeout time.Duration
	Retries int
}

// NewEngine builds an engine with sane defaults.
func NewEngine() *Engine {
	return &Engine{Port: 161, Timeout: 2 * time.Second, Retries: 1}
}

func (e *Engine) conn(target string, cred Credential) (*g.GoSNMP, error) {
	port := e.Port
	if port == 0 {
		port = 161
	}
	c := &g.GoSNMP{
		Target:  target,
		Port:    port,
		Timeout: e.Timeout,
		Retries: e.Retries,
	}
	switch cred.Version {
	case V3:
		c.Version = g.Version3
		c.SecurityModel = g.UserSecurityModel
		c.MsgFlags = g.AuthPriv
		c.SecurityParameters = &g.UsmSecurityParameters{
			UserName:                 cred.User,
			AuthenticationProtocol:   authProto(cred.AuthProto),
			AuthenticationPassphrase: cred.AuthPass,
			PrivacyProtocol:          privProto(cred.PrivProto),
			PrivacyPassphrase:        cred.PrivPass,
		}
	default:
		c.Version = g.Version2c
		c.Community = cred.Community
		if c.Community == "" {
			c.Community = "public"
		}
	}
	if err := c.Connect(); err != nil {
		return nil, fmt.Errorf("snmp connect %s: %w", target, err)
	}
	return c, nil
}

// Get fetches the given OIDs and returns a map of OID -> value.
func (e *Engine) Get(target string, cred Credential, oids ...string) (map[string]any, error) {
	c, err := e.conn(target, cred)
	if err != nil {
		return nil, err
	}
	defer c.Conn.Close()
	res, err := c.Get(oids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(res.Variables))
	for _, v := range res.Variables {
		out[v.Name] = decodePDU(v)
	}
	return out, nil
}

func decodePDU(v g.SnmpPDU) any {
	switch v.Type {
	case g.OctetString:
		if b, ok := v.Value.([]byte); ok {
			return string(b)
		}
	case g.Counter64, g.Counter32, g.Gauge32, g.Integer, g.TimeTicks, g.Uinteger32:
		return g.ToBigInt(v.Value).Uint64()
	}
	return v.Value
}

func authProto(s string) g.SnmpV3AuthProtocol {
	switch s {
	case "MD5":
		return g.MD5
	default:
		return g.SHA
	}
}

func privProto(s string) g.SnmpV3PrivProtocol {
	switch s {
	case "DES":
		return g.DES
	default:
		return g.AES
	}
}
