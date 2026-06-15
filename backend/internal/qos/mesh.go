package qos

import (
	"github.com/neko/sdwan/backend/internal/configengine"
	"github.com/neko/sdwan/backend/internal/routing"
	"github.com/neko/sdwan/backend/internal/store"
)

// ApplyToMeshPlan merges per-site rate limits into CPE node desired states.
func ApplyToMeshPlan(plan *routing.MeshPlan, sites []routing.MeshSiteInput, devices map[string]*store.Device) error {
	byID := make(map[string]*routing.MeshNodePlan, len(plan.Nodes))
	for i := range plan.Nodes {
		byID[plan.Nodes[i].DeviceID] = &plan.Nodes[i]
	}
	for _, site := range sites {
		if site.RateLimit == "" {
			continue
		}
		node, ok := byID[site.DeviceID]
		if !ok {
			continue
		}
		name := site.DeviceID
		if d := devices[site.DeviceID]; d != nil {
			name = d.Name
		}
		rules, err := RulesForSite(name, site.Prefixes, site.RateLimit, site.RateTarget)
		if err != nil {
			return err
		}
		desired, err := MergeState(node.Desired, rules)
		if err != nil {
			return err
		}
		node.Desired = desired
		node.Plan = configengine.ComputeDiff(configengine.State{}, desired, configengine.RiskOptions{})
	}
	return nil
}
