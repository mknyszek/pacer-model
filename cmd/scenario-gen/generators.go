package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/mknyszek/pacer-model/scenario"
)

func Generate(name string) (scenario.Execution, error) {
	g, ok := generators[name]
	if !ok {
		return scenario.Execution{}, fmt.Errorf("generator %q not found", name)
	}
	return generate(g()), nil
}

func Generators() []string {
	var s []string
	for name := range generators {
		s = append(s, name)
	}
	sort.Strings(s)
	return s
}

func generate(e exec) scenario.Execution {
	c := make([]scenario.Cycle, 0, e.length)
	for i := 0; i < e.length; i++ {
		c = append(c, scenario.Cycle{
			AllocRate:     e.allocRate.min(0)(),
			ScanRate:      e.scanRate.min(0)(),
			GrowthRate:    e.growthRate.min(0)(),
			ScannableFrac: e.scannableFrac.limit(0, 1)(),
			StackBytes:    uint64(e.stackBytes.quantize(2048).min(0)()),
		})
	}
	return scenario.Execution{
		Globals: e.globals,
		Cycles:  c,
	}
}

type exec struct {
	globals scenario.Globals

	allocRate     stream
	scanRate      stream
	growthRate    stream
	scannableFrac stream
	stackBytes    stream
	length        int
}

var generators = map[string]func() exec{
	"steady": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     constant(1.0),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        50,
		}
	},
	"step-alloc": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     constant(1.0).mix(ramp(1.0, 1).delay(50)),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        100,
		}
	},
	"big-stacks": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     constant(4.0),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(2048).mix(ramp(128<<20, 8)),
			length:        50,
		}
	},
	"big-globals": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 128 << 20,
				InitialHeap:  2 << 20,
			},
			allocRate:     constant(4.0),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        50,
		}
	},
	"osc-alloc": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     oscillate(0.4, 0, 8).offset(2),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        50,
		}
	},
	"jitter-alloc": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     random(0.4).offset(4),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8), random(0.01)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        50,
		}
	},
	"high-GOGC": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        8,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     random(0.2).offset(5),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8), random(0.01), unit(7).delay(25)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        50,
		}
	},
	"heavy-alloc": func() exec {
		return exec{
			globals: scenario.Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:     random(1.0).offset(10),
			scanRate:      constant(31.0),
			growthRate:    constant(2.0).mix(ramp(-1.0, 8), random(0.01)),
			scannableFrac: constant(1.0),
			stackBytes:    constant(8192),
			length:        100,
		}
	},
}

type stream func() float64

func constant(c float64) stream {
	return func() float64 {
		return c
	}
}

func unit(amp float64) stream {
	dropped := false
	return func() float64 {
		if dropped {
			return 0
		}
		dropped = true
		return amp
	}
}

func oscillate(amp, phase float64, period int) stream {
	var cycle int
	return func() float64 {
		p := float64(cycle)/float64(period)*2*math.Pi + phase
		cycle++
		if cycle == period {
			cycle = 0
		}
		return math.Sin(p) * amp
	}
}

func ramp(height float64, length int) stream {
	var cycle int
	return func() float64 {
		h := height * float64(cycle) / float64(length)
		if cycle < length {
			cycle++
		}
		return h
	}
}

func random(amp float64) stream {
	return func() float64 {
		return ((rand.Float64() - 0.5) * 2) * amp
	}
}

func (f stream) delay(cycles int) stream {
	buf := make([]float64, 0, cycles)
	next := 0
	return func() float64 {
		old := f()
		if len(buf) < cap(buf) {
			buf = append(buf, old)
			return 0
		}
		res := buf[next]
		buf[next] = old
		next++
		if next == len(buf) {
			next = 0
		}
		return res
	}
}

func (f stream) vga(gain stream) stream {
	return func() float64 {
		return f() * gain()
	}
}

func (f stream) scale(amt float64) stream {
	return f.vga(constant(amt))
}

func (f stream) offset(amt float64) stream {
	return func() float64 {
		old := f()
		return old + amt
	}
}

func (f stream) mix(fs ...stream) stream {
	return func() float64 {
		sum := f()
		for _, s := range fs {
			sum += s()
		}
		return sum
	}
}

func (f stream) quantize(mult float64) stream {
	return func() float64 {
		r := f() / mult
		if r < 0 {
			return math.Ceil(r) * mult
		}
		return math.Floor(r) * mult
	}
}

func (f stream) min(min float64) stream {
	return func() float64 {
		return math.Max(min, f())
	}
}

func (f stream) max(max float64) stream {
	return func() float64 {
		return math.Min(max, f())
	}
}

func (f stream) limit(min, max float64) stream {
	return func() float64 {
		v := f()
		if v < min {
			v = min
		} else if v > max {
			v = max
		}
		return v
	}
}
