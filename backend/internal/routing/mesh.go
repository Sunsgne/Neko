package routing

import (
	"fmt"
	"sort"
	"strings"

	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/store"
)

// MeshTopology describes how sites interconnect across backbone nodes.
type MeshTopology string

const (
	// MeshHubSpoke: each CPE homes on a POP; POPs interconnect (iBGP) to reach remote sites.
	MeshHubSpoke MeshTopology = "hub_spoke"
	// MeshTransit: explicit POP chain — Device → POP₁ → POP₂ → … → Device.
	MeshTransit MeshTopology = "transit"
	// MeshFullMesh: every backbone node peers with every other (WG + iBGP).
	MeshFullMesh MeshTopology = "full_mesh"
)

// MeshSiteInput is one customer site attached to a home POP.
type MeshSiteInput struct {
	DeviceID    string   `json:"device_id"`
	PopDeviceID string   `json:"pop_device_id"`
	Prefixes    []string `json:"prefixes"`
	CpeOverlay  string   `json:"cpe_overlay,omitempty"`
	RateLimit   string   `json:"rate_limit,omitempty"`  // e.g. 10M — applied per site on deploy
	RateTarget  string   `json:"rate_target,omitempty"` // optional override; defaults to prefixes
}

// MeshPlanInput is the full mesh / transit fabric request.
type MeshPlanInput struct {
	Topology     MeshTopology    `json:"topology"`
	LocalAS      uint32          `json:"local_as"`
	Sites        []MeshSiteInput `json:"sites"`
	BackbonePath []string        `json:"backbone_path,omitempty"` // ordered POP IDs (transit / full_mesh)
	RRDeviceID   string          `json:"rr_device_id,omitempty"` // optional route-reflector backbone
}

// MeshLink summarizes one overlay adjacency in the plan.
type MeshLink struct {
	Kind         string `json:"kind"` // cpe_pop | pop_pop
	FromDeviceID string `json:"from_device_id"`
	ToDeviceID   string `json:"to_device_id"`
	FromIface    string `json:"from_iface,omitempty"`
	ToIface      string `json:"to_iface,omitempty"`
	FromOverlay  string `json:"from_overlay,omitempty"`
	ToOverlay    string `json:"to_overlay,omitempty"`
	ToGateway    string `json:"to_gateway,omitempty"`
}

// MeshNodePlan is the generated config for one device in the fabric.
type MeshNodePlan struct {
	DeviceID   string              `json:"device_id"`
	DeviceName string              `json:"device_name"`
	Role       string              `json:"role"`
	Desired    configengine.State  `json:"desired"`
	Plan       configengine.Plan   `json:"plan"`
}

// MeshPlan is the multi-site SD-WAN fabric output.
type MeshPlan struct {
	Topology MeshTopology   `json:"topology"`
	LocalAS  uint32         `json:"local_as"`
	Links    []MeshLink     `json:"links"`
	Nodes    []MeshNodePlan `json:"nodes"`
	Summary  string         `json:"summary"`
}

// BuildMeshPlan composes WireGuard overlays and iBGP routing for multi-site
// hub-spoke, transit (D→B→B→D), or backbone full-mesh topologies.
func BuildMeshPlan(devices map[string]*store.Device, in MeshPlanInput) (MeshPlan, error) {
	if len(in.Sites) < 2 {
		return MeshPlan{}, fmt.Errorf("mesh 至少需要 2 个站点")
	}
	if in.LocalAS == 0 {
		in.LocalAS = 65001
	}
	topology := in.Topology
	if topology == "" {
		topology = MeshHubSpoke
	}

	for i, s := range in.Sites {
		if s.DeviceID == "" || s.PopDeviceID == "" {
			return MeshPlan{}, fmt.Errorf("sites[%d]: device_id and pop_device_id required", i)
		}
		if len(s.Prefixes) == 0 {
			return MeshPlan{}, fmt.Errorf("sites[%d]: 至少宣告一条内网前缀", i)
		}
		cpe := devices[s.DeviceID]
		pop := devices[s.PopDeviceID]
		if cpe == nil || pop == nil {
			return MeshPlan{}, fmt.Errorf("sites[%d]: device or pop not found", i)
		}
		if isBackbone(cpe) {
			return MeshPlan{}, fmt.Errorf("sites[%d]: device %s 必须是 CPE", i, cpe.Name)
		}
		if !isBackbone(pop) {
			return MeshPlan{}, fmt.Errorf("sites[%d]: pop %s 必须是骨干/出口节点", i, pop.Name)
		}
	}

	popIDs := uniquePopIDs(in.Sites)
	backbonePairs, err := backbonePairsForTopology(topology, popIDs, in.BackbonePath)
	if err != nil {
		return MeshPlan{}, err
	}

	desiredByID := make(map[string]configengine.State)
	links := []MeshLink{}
	cpeLinks := map[string]WGLink{} // cpeID -> link to home pop

	// CPE ↔ home POP tunnels
	for _, site := range in.Sites {
		cpe := devices[site.DeviceID]
		pop := devices[site.PopDeviceID]
		link, err := BuildWGLink(cpe, pop, "", "")
		if err != nil {
			return MeshPlan{}, err
		}
		if site.CpeOverlay != "" {
			if err := applyCPEOverlay(&link, site.CpeOverlay); err != nil {
				return MeshPlan{}, err
			}
		}
		cpeLinks[site.DeviceID] = link

		links = append(links, MeshLink{
			Kind:         "cpe_pop",
			FromDeviceID: cpe.ID,
			ToDeviceID:   pop.ID,
			FromIface:    link.CPEInterface,
			ToIface:      link.POPInterface,
			FromOverlay:  link.CPEOverlay,
			ToOverlay:    link.POPOverlay,
			ToGateway:    link.POPGateway,
		})

		cpeState := link.CPEState()
		cpeState = configengine.Merge(cpeState, meshSiteBGP(cpe.ID, link.POPGateway, in.LocalAS, site.Prefixes, false))
		desiredByID[cpe.ID] = configengine.Merge(desiredByID[cpe.ID], cpeState)

		popPeer := popCPEPeerState(link, site.Prefixes)
		popBGP := meshSiteBGP(pop.ID+"-"+cpe.ID, link.CPEOverlayHost(), in.LocalAS, site.Prefixes, false)
		desiredByID[pop.ID] = configengine.Merge(desiredByID[pop.ID], popPeer, popBGP)
	}

	// POP ↔ POP backbone links
	popPopLinks := map[string]WGLink{} // "a:b" -> link (a < b)
	for _, pair := range backbonePairs {
		popA := devices[pair[0]]
		popB := devices[pair[1]]
		if popA == nil || popB == nil {
			return MeshPlan{}, fmt.Errorf("backbone pair device not found")
		}
		// Stable ordering for overlay allocation
		left, right := popA, popB
		if left.ID > right.ID {
			left, right = right, left
		}
		link, err := BuildWGLink(left, right, "", "")
		if err != nil {
			return MeshPlan{}, err
		}
		key := left.ID + ":" + right.ID
		popPopLinks[key] = link

		links = append(links, MeshLink{
			Kind:         "pop_pop",
			FromDeviceID: left.ID,
			ToDeviceID:   right.ID,
			FromIface:    link.CPEInterface,
			ToIface:      link.POPInterface,
			FromOverlay:  link.CPEOverlay,
			ToOverlay:    link.POPOverlay,
			ToGateway:    link.POPGateway,
		})

		// left side (uses CPE-style endpoint naming)
		leftState := backboneEndpointState(link, true)
		rightGW := link.POPGateway
		leftRR := in.RRDeviceID == left.ID
		leftBGP := meshSiteBGP("bb-"+right.ID, rightGW, in.LocalAS, nil, leftRR)
		desiredByID[left.ID] = configengine.Merge(desiredByID[left.ID], leftState, leftBGP)

		// right side — mirror link from B's perspective
		mirror := mirrorBackboneLink(link, right, left)
		rightState := backboneEndpointState(mirror, true)
		leftGW := PopPeerOf(link.CPEOverlay)
		rightRR := in.RRDeviceID == right.ID
		rightBGP := meshSiteBGP("bb-"+left.ID, leftGW, in.LocalAS, nil, rightRR)
		desiredByID[right.ID] = configengine.Merge(desiredByID[right.ID], rightState, rightBGP)
	}

	// Remote site reachability: static fallback routes on CPE via home POP
	for _, site := range in.Sites {
		link := cpeLinks[site.DeviceID]
		homePop := site.PopDeviceID
		var remoteRoutes []string
		for _, other := range in.Sites {
			if other.DeviceID == site.DeviceID {
				continue
			}
			remoteRoutes = append(remoteRoutes, other.Prefixes...)
		}
		if len(remoteRoutes) > 0 {
			desiredByID[site.DeviceID] = configengine.Merge(
				desiredByID[site.DeviceID],
				meshReachabilityRoutes(remoteRoutes, "", link.POPGateway),
			)
		}
		// POP static routes to remote prefixes via backbone next-hop
		for _, other := range in.Sites {
			if other.PopDeviceID == homePop || other.DeviceID == site.DeviceID {
				continue
			}
			nh := backboneNextHop(homePop, other.PopDeviceID, popPopLinks)
			if nh != "" {
				desiredByID[homePop] = configengine.Merge(
					desiredByID[homePop],
					meshReachabilityRoutes(other.Prefixes, "", nh),
				)
			}
		}
	}

	nodes := make([]MeshNodePlan, 0, len(desiredByID))
	for id, st := range desiredByID {
		dev := devices[id]
		name, role := id, "cpe"
		if dev != nil {
			name = dev.Name
			role = string(dev.Role)
		}
		nodes = append(nodes, MeshNodePlan{
			DeviceID:   id,
			DeviceName: name,
			Role:       role,
			Desired:    st,
			Plan:       configengine.ComputeDiff(configengine.State{}, st, configengine.RiskOptions{}),
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		ri, rj := roleRank(nodes[i].Role), roleRank(nodes[j].Role)
		if ri != rj {
			return ri < rj
		}
		return nodes[i].DeviceName < nodes[j].DeviceName
	})

	return MeshPlan{
		Topology: topology,
		LocalAS:  in.LocalAS,
		Links:    links,
		Nodes:    nodes,
		Summary:  describeMesh(topology, len(in.Sites), len(backbonePairs)),
	}, nil
}

func describeMesh(t MeshTopology, sites, bbLinks int) string {
	switch t {
	case MeshTransit:
		return fmt.Sprintf("transit 组网：%d 个站点，%d 段骨干互联", sites, bbLinks)
	case MeshFullMesh:
		return fmt.Sprintf("骨干 full mesh：%d 个站点，%d 条骨干全互联", sites, bbLinks)
	default:
		return fmt.Sprintf("hub-spoke 组网：%d 个站点，%d 条骨干互联", sites, bbLinks)
	}
}

func roleRank(role string) int {
	switch role {
	case "backbone", "gateway":
		return 0
	default:
		return 1
	}
}

func isBackbone(d *store.Device) bool {
	return d.Role == "backbone" || d.Role == "gateway"
}

func uniquePopIDs(sites []MeshSiteInput) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range sites {
		if _, ok := seen[s.PopDeviceID]; !ok {
			seen[s.PopDeviceID] = struct{}{}
			out = append(out, s.PopDeviceID)
		}
	}
	sort.Strings(out)
	return out
}

func backbonePairsForTopology(topology MeshTopology, popIDs, path []string) ([][2]string, error) {
	switch topology {
	case MeshTransit:
		if len(path) < 2 {
			return nil, fmt.Errorf("transit 拓扑需要 backbone_path 至少 2 个骨干节点")
		}
		var pairs [][2]string
		for i := 0; i < len(path)-1; i++ {
			pairs = append(pairs, [2]string{path[i], path[i+1]})
		}
		return pairs, nil
	case MeshFullMesh:
		ids := path
		if len(ids) < 2 {
			ids = popIDs
		}
		if len(ids) < 2 {
			return nil, fmt.Errorf("full_mesh 至少需要 2 个骨干节点")
		}
		var pairs [][2]string
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				pairs = append(pairs, [2]string{ids[i], ids[j]})
			}
		}
		return pairs, nil
	default: // hub_spoke — connect all involved POPs
		if len(popIDs) < 2 {
			return nil, nil
		}
		var pairs [][2]string
		for i := 0; i < len(popIDs); i++ {
			for j := i + 1; j < len(popIDs); j++ {
				pairs = append(pairs, [2]string{popIDs[i], popIDs[j]})
			}
		}
		return pairs, nil
	}
}

func popCPEPeerState(link WGLink, sitePrefixes []string) configengine.State {
	allowed := allowedForSite(link.CPEOverlay, sitePrefixes)
	return buildWGEndpoint(wgEndpoint{
		iface:       link.POPInterface,
		privateKey:  link.POPPrivateKey,
		listenPort:  link.ListenPort,
		overlayAddr: link.POPOverlay,
		peerKey:     link.CPEPublicKey,
		peerHost:    link.CPEEndpointHost,
		peerPort:    link.ListenPort,
		peerName:    link.POPInterface + "-peer-" + link.CPEInterface,
		allowed:     allowed,
	})
}

func allowedForSite(cpeOverlay string, prefixes []string) string {
	host, _ := splitHostPrefix(cpeOverlay)
	parts := append([]string{host + "/32"}, prefixes...)
	return strings.Join(parts, ",")
}

func (l WGLink) CPEOverlayHost() string {
	host, _ := splitHostPrefix(l.CPEOverlay)
	return host
}

func backboneEndpointState(link WGLink, hub bool) configengine.State {
	allowed := "0.0.0.0/0"
	if hub {
		allowed = "100.64.0.0/16,0.0.0.0/0"
	}
	return buildWGEndpoint(wgEndpoint{
		iface:       link.CPEInterface,
		privateKey:  link.CPEPrivateKey,
		listenPort:  link.ListenPort,
		overlayAddr: link.CPEOverlay,
		peerKey:     link.POPPublicKey,
		peerHost:    link.POPEndpointHost,
		peerPort:    link.ListenPort,
		peerName:    link.CPEInterface + "-peer",
		allowed:     allowed,
	})
}

func mirrorBackboneLink(link WGLink, local, remote *store.Device) WGLink {
	m := link
	m.CPEInterface = TunnelNameForPOP(local.Name) + "-bb"
	if len(m.CPEInterface) > 24 {
		m.CPEInterface = m.CPEInterface[:24]
	}
	m.POPInterface = tunnelNameForCPE(remote.Name)
	m.CPEOverlay, m.POPOverlay = link.POPOverlay, link.CPEOverlay
	m.POPGateway = PopPeerOf(link.POPOverlay)
	m.CPEPrivateKey, m.POPPrivateKey = link.POPPrivateKey, link.CPEPrivateKey
	m.CPEPublicKey, m.POPPublicKey = link.POPPublicKey, link.CPEPublicKey
	m.CPEEndpointHost, m.POPEndpointHost = link.POPEndpointHost, link.CPEEndpointHost
	return m
}

func meshSiteBGP(sessionName, peerAddr string, localAS uint32, advertise []string, rrClient bool) configengine.State {
	peer := peerAddr
	if i := strings.Index(peer, "/"); i > 0 {
		peer = peer[:i]
	}
	intent := Intent{
		BGP: []BGPNeighbor{{
			Name:        "neko-" + sessionName,
			PeerAddress: peer,
			LocalAS:     localAS,
			PeerAS:      localAS,
			RRClient:    rrClient,
			BFD:         true,
		}},
	}
	st := BuildState(intent)
	for _, p := range advertise {
		st = configengine.Merge(st, configengine.State{Statements: []configengine.Statement{
			{
				Path: "/routing/bgp/aggregate", Key: p,
				Attributes: map[string]string{"prefix": p, "advertise": "yes"},
			},
		}})
	}
	return st
}

func backboneNextHop(fromPop, toPop string, links map[string]WGLink) string {
	if fromPop == toPop {
		return ""
	}
	a, b := fromPop, toPop
	if a > b {
		a, b = b, a
	}
	link, ok := links[a+":"+b]
	if !ok {
		return ""
	}
	if fromPop == a {
		return link.POPGateway
	}
	return PopPeerOf(link.CPEOverlay)
}

// MeshDeployOrder returns device IDs in safe apply order: backbones first, then CPEs.
func MeshDeployOrder(nodes []MeshNodePlan) []string {
	var bb, cpe []string
	for _, n := range nodes {
		if n.Role == "backbone" || n.Role == "gateway" {
			bb = append(bb, n.DeviceID)
		} else {
			cpe = append(cpe, n.DeviceID)
		}
	}
	return append(bb, cpe...)
}
