package simulation

import (
	"fmt"
	"sort"

	"github.com/mknyszek/pacer-model/controller"
	"github.com/mknyszek/pacer-model/scenario"
)

type Simulator interface {
	Step(*scenario.Cycle) Result
}

type simFactory func(scenario.Globals, controller.Controller) Simulator

var sims = map[string]simFactory{
	"go116": func(g scenario.Globals, _ controller.Controller) Simulator {
		return &go116{Globals: g}
	},
	"go117": func(g scenario.Globals, c controller.Controller) Simulator {
		if c == nil {
			c = controller.NewPI(&controller.PIConfig{
				Kp:     0.9,
				Ti:     1.6,
				Tt:     1000,
				Period: 1,
				Min:    -2,
				Max:    2,
			})
		}
		return &go117{Globals: g, ctrl: c}
	},
}

func Simulators() []string {
	var s []string
	for name := range sims {
		s = append(s, name)
	}
	sort.Strings(s)
	return s
}

func NewSimulator(name string, globals scenario.Globals, ctrl controller.Controller) (Simulator, error) {
	f, ok := sims[name]
	if !ok {
		return nil, fmt.Errorf("unknown pacer type %q", name)
	}
	return f(globals, ctrl), nil
}

type Result struct {
	R                   float64 `json:"r"`
	LiveBytes           uint64  `json:"live"`
	LiveScanBytes       uint64  `json:"scan"`
	GoalBytes           uint64  `json:"goal"`
	ActualGCUtilization float64 `json:"actual_u"`
	TargetGCUtilization float64 `json:"target_u"`
	TriggerPoint        uint64  `json:"trigger"`
	PeakBytes           uint64  `json:"peak"`
}
