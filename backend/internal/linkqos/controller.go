package linkqos

import (
	"sort"
	"time"
)

// Role is the configured preference of a link.
type Role string

const (
	RolePrimary Role = "primary"
	RoleBackup  Role = "backup"
)

func roleRank(r Role) int {
	if r == RolePrimary {
		return 0
	}
	return 1
}

// LinkInput is the per-observation state of a candidate link.
type LinkInput struct {
	ID    string
	Role  Role
	Score float64
}

// Thresholds configures the flap-damping state machine. Hysteresis is enforced
// by keeping UpScore > DownScore, plus minimum durations and a dwell window.
type Thresholds struct {
	DownScore float64       // below this, a link is trending bad
	UpScore   float64       // at/above this, a link is trending good
	MinDown   time.Duration // active link must be bad this long before failover
	MinUp     time.Duration // a link must be good this long before it is eligible
	MinDwell  time.Duration // minimum time between switches (anti-oscillation)
}

// DefaultThresholds returns conservative anti-flap defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{
		DownScore: 40,
		UpScore:   70,
		MinDown:   15 * time.Second,
		MinUp:     30 * time.Second,
		MinDwell:  60 * time.Second,
	}
}

// Decision is the outcome of an observation.
type Decision struct {
	Active   string
	Switched bool
	Reason   string
}

// Controller tracks per-link timers and decides the active link over time.
// It is not safe for concurrent use; callers should serialize Observe calls
// per controlled link group.
type Controller struct {
	thr        Thresholds
	active     string
	lastSwitch time.Time
	started    bool
	badSince   map[string]time.Time
	goodSince  map[string]time.Time
}

// NewController builds a controller with the given thresholds.
func NewController(thr Thresholds) *Controller {
	return &Controller{
		thr:       thr,
		badSince:  map[string]time.Time{},
		goodSince: map[string]time.Time{},
	}
}

// Active returns the currently selected link id.
func (c *Controller) Active() string { return c.active }

// Observe ingests the current scores and returns the active link decision.
func (c *Controller) Observe(now time.Time, inputs []LinkInput) Decision {
	byID := make(map[string]LinkInput, len(inputs))
	for _, in := range inputs {
		byID[in.ID] = in
	}
	c.updateTimers(now, inputs, byID)

	// Initialize on first observation.
	if !c.started {
		best := c.pickEligible(now, inputs)
		if best == "" {
			best = c.pickBest(inputs)
		}
		c.active = best
		c.lastSwitch = now
		c.started = true
		return Decision{Active: c.active, Switched: best != "", Reason: "init"}
	}

	// Dwell window: suppress switching to avoid oscillation.
	if now.Sub(c.lastSwitch) < c.thr.MinDwell {
		return Decision{Active: c.active, Switched: false, Reason: "dwell"}
	}

	best := c.pickEligible(now, inputs)
	activeIn, activePresent := byID[c.active]
	activeBad := !activePresent || c.confirmedBad(now, c.active)

	// Failover: active is bad and a healthy alternative exists.
	if activeBad && best != "" && best != c.active {
		return c.switchTo(now, best, "failover")
	}

	// Failback / preference: a higher-priority healthy link is available.
	if best != "" && best != c.active && activePresent {
		bestIn := byID[best]
		if roleRank(bestIn.Role) < roleRank(activeIn.Role) {
			return c.switchTo(now, best, "failback")
		}
	}

	return Decision{Active: c.active, Switched: false, Reason: "stable"}
}

func (c *Controller) switchTo(now time.Time, id, reason string) Decision {
	c.active = id
	c.lastSwitch = now
	return Decision{Active: id, Switched: true, Reason: reason}
}

func (c *Controller) updateTimers(now time.Time, inputs []LinkInput, byID map[string]LinkInput) {
	for _, in := range inputs {
		switch {
		case in.Score < c.thr.DownScore:
			if _, ok := c.badSince[in.ID]; !ok {
				c.badSince[in.ID] = now
			}
			delete(c.goodSince, in.ID)
		case in.Score >= c.thr.UpScore:
			if _, ok := c.goodSince[in.ID]; !ok {
				c.goodSince[in.ID] = now
			}
			delete(c.badSince, in.ID)
		}
		// In the hysteresis band [DownScore, UpScore): keep existing timers.
	}
	// Drop timers for links no longer present.
	for id := range c.badSince {
		if _, ok := byID[id]; !ok {
			delete(c.badSince, id)
		}
	}
	for id := range c.goodSince {
		if _, ok := byID[id]; !ok {
			delete(c.goodSince, id)
		}
	}
}

func (c *Controller) confirmedBad(now time.Time, id string) bool {
	t, ok := c.badSince[id]
	return ok && now.Sub(t) >= c.thr.MinDown
}

func (c *Controller) confirmedGood(now time.Time, id string) bool {
	t, ok := c.goodSince[id]
	return ok && now.Sub(t) >= c.thr.MinUp
}

// pickEligible returns the best link that is confirmed-good, preferring lower
// role rank (primary) then higher score.
func (c *Controller) pickEligible(now time.Time, inputs []LinkInput) string {
	eligible := make([]LinkInput, 0, len(inputs))
	for _, in := range inputs {
		if c.confirmedGood(now, in.ID) {
			eligible = append(eligible, in)
		}
	}
	return rankPick(eligible)
}

func (c *Controller) pickBest(inputs []LinkInput) string {
	cp := append([]LinkInput(nil), inputs...)
	return rankPick(cp)
}

func rankPick(in []LinkInput) string {
	if len(in) == 0 {
		return ""
	}
	sort.SliceStable(in, func(i, j int) bool {
		ri, rj := roleRank(in[i].Role), roleRank(in[j].Role)
		if ri != rj {
			return ri < rj
		}
		if in[i].Score != in[j].Score {
			return in[i].Score > in[j].Score
		}
		return in[i].ID < in[j].ID
	})
	return in[0].ID
}
