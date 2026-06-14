package configengine

import "sort"

// ComputeDiff returns the minimal set of changes that converges running toward
// desired. Statements present in running but absent from desired are removed;
// new statements are added; statements present in both with differing
// attributes are updated at the attribute level.
//
// The resulting plan is deterministically ordered (by path, then key, then
// change type) so repeated runs produce identical plans.
func ComputeDiff(running, desired State, opts RiskOptions) Plan {
	runIdx := running.index()
	desIdx := desired.index()

	var changes []Change

	// Adds and updates.
	for _, d := range desired.Statements {
		k := d.Path + "\x00" + d.Key
		r, ok := runIdx[k]
		if !ok {
			changes = append(changes, Change{Type: ChangeAdd, Path: d.Path, Key: d.Key, Attrs: attrsAsChanges(d.Attributes)})
			continue
		}
		if ac := attrDiff(r.Attributes, d.Attributes); len(ac) > 0 {
			changes = append(changes, Change{
				Type:  ChangeUpdate,
				Path:  d.Path,
				Key:   d.Key,
				Attrs: sortedAttrChanges(ac),
			})
		}
	}

	// Removals.
	for _, r := range running.Statements {
		k := r.Path + "\x00" + r.Key
		if _, ok := desIdx[k]; !ok {
			changes = append(changes, Change{Type: ChangeRemove, Path: r.Path, Key: r.Key})
		}
	}

	// Deterministic ordering.
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Path != changes[j].Path {
			return changes[i].Path < changes[j].Path
		}
		if changes[i].Key != changes[j].Key {
			return changes[i].Key < changes[j].Key
		}
		return changes[i].Type < changes[j].Type
	})

	// Classify risk per change and aggregate.
	agg := RiskLow
	for i := range changes {
		changes[i].Risk, changes[i].Reason = classifyChange(changes[i], opts)
		if riskRank(changes[i].Risk) > riskRank(agg) {
			agg = changes[i].Risk
		}
	}

	return Plan{Changes: changes, AggregateRisk: agg}
}

// attrsAsChanges renders a full attribute set as AttrChanges (for adds).
func attrsAsChanges(attrs map[string]string) []AttrChange {
	out := make([]AttrChange, 0, len(attrs))
	for k, v := range attrs {
		out = append(out, AttrChange{Attr: k, Old: "", New: v})
	}
	return sortedAttrChanges(out)
}

func attrDiff(old, new map[string]string) []AttrChange {
	var out []AttrChange
	for k, nv := range new {
		if ov, ok := old[k]; !ok || ov != nv {
			out = append(out, AttrChange{Attr: k, Old: old[k], New: nv})
		}
	}
	for k, ov := range old {
		if _, ok := new[k]; !ok {
			out = append(out, AttrChange{Attr: k, Old: ov, New: ""})
		}
	}
	return out
}
