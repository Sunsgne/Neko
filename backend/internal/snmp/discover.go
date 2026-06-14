package snmp

import (
	"context"
	"net"
	"sync"
)

// Discovered is a device found during a scan.
type Discovered struct {
	Address     string `json:"address"`
	SysDescr    string `json:"sys_descr"`
	SysObjectID string `json:"sys_object_id"`
	SysName     string `json:"sys_name"`
}

// Discover scans a CIDR, probing each host with an SNMP GET of system OIDs.
// Hosts that respond are returned. Concurrency is bounded by workers.
func (e *Engine) Discover(ctx context.Context, cidr string, cred Credential, workers int) ([]Discovered, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	if workers <= 0 {
		workers = 32
	}

	hosts := make(chan string)
	go func() {
		defer close(hosts)
		for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			select {
			case <-ctx.Done():
				return
			case hosts <- ip.String():
			}
		}
	}()

	var (
		mu    sync.Mutex
		found []Discovered
		wg    sync.WaitGroup
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range hosts {
				if ctx.Err() != nil {
					return
				}
				res, err := e.Get(host, cred, OIDsysDescr, OIDsysObjectID, OIDsysName)
				if err != nil {
					continue
				}
				d := Discovered{
					Address:     host,
					SysDescr:    asString(res[OIDsysDescr]),
					SysObjectID: asString(res[OIDsysObjectID]),
					SysName:     asString(res[OIDsysName]),
				}
				if d.SysDescr == "" && d.SysObjectID == "" {
					continue
				}
				mu.Lock()
				found = append(found, d)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return found, ctx.Err()
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
