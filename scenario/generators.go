package scenario

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

func Generate(name string) (Execution, error) {
	g, ok := generators[name]
	if !ok {
		return Execution{}, fmt.Errorf("generator %q not found", name)
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

func generate(e exec) Execution {
	c := make([]Cycle, 0, e.length)
	for i := 0; i < e.length; i++ {
		c = append(c, Cycle{
			AllocRate:       e.allocRate.min(0)(),
			ScanRate:        e.scanRate.min(0)(),
			GrowthRate:      e.growthRate.min(0)(),
			ScannableFrac:   e.scannableFrac.limit(0, 1)(),
			StackBytes:      uint64(e.stackBytes.quantize(2048).min(0)()),
			HeapTargetBytes: int64(e.heapTargetBytes.quantize(1)()),
		})
	}
	return Execution{
		Globals: e.globals,
		Cycles:  c,
	}
}

type exec struct {
	globals Globals

	allocRate       stream
	scanRate        stream
	growthRate      stream
	scannableFrac   stream
	stackBytes      stream
	heapTargetBytes stream
	length          int
}

var generators = map[string]func() exec{
	"steady": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       constant(1.0),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"step-alloc": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       constant(1.0).mix(ramp(1.0, 1).delay(50)),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          100,
		}
	},
	"big-stacks": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       constant(4.0),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(2048).mix(ramp(128<<20, 8)),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"big-globals": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 128 << 20,
				InitialHeap:  2 << 20,
			},
			allocRate:       constant(4.0),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"osc-alloc": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       oscillate(0.4, 0, 8).offset(2),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"jitter-alloc": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.4).offset(4),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"high-GOGC": func() exec {
		return exec{
			globals: Globals{
				Gamma:        16,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.2).offset(5),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01), unit(14).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"heavy-jitter-alloc": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(1.0).offset(10),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          50,
		}
	},
	"heavy-step-alloc": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       constant(1.0).mix(ramp(10.0, 1).delay(50)),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1),
			length:          100,
		}
	},
	"high-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.2).offset(5),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01), unit(14).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(2 << 30),
			length:          50,
		}
	},
	"low-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.1).offset(4),
			scanRate:        constant(31.0),
			growthRate:      constant(1.5).mix(ramp(-0.5, 4), random(0.01), unit(3).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(64 << 20),
			length:          50,
		}
	},
	"very-low-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.1).offset(4),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 20), random(0.01)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(64 << 20),
			length:          50,
		}
	},
	"step-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.1).offset(4),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(-1).mix(constant((256 << 20) + 1).delay(25)),
			length:          50,
		}
	},
	"heavy-step-alloc-high-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       constant(1.0).mix(ramp(10.0, 1).delay(25)),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(2 << 30),
			length:          50,
		}
	},
	"exceed-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.1).offset(4),
			scanRate:        constant(31.0),
			growthRate:      constant(1.5).mix(ramp(-0.5, 4), random(0.01), unit(6).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(64 << 20),
			length:          50,
		}
	},
	"exceed-heap-target-high-GOGC": func() exec {
		return exec{
			globals: Globals{
				Gamma:        16,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.1).offset(4),
			scanRate:        constant(31.0),
			growthRate:      constant(1.5).mix(ramp(-0.5, 4), random(0.01), unit(14).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(64 << 20),
			length:          50,
		}
	},
	"low-noise-high-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.2).offset(5),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01), unit(14).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(2 << 30).mix(random(1 << 20)),
			length:          50,
		}
	},
	"high-noise-high-heap-target": func() exec {
		return exec{
			globals: Globals{
				Gamma:        2,
				GlobalsBytes: 32 << 10,
				InitialHeap:  2 << 20,
			},
			allocRate:       random(0.2).offset(5),
			scanRate:        constant(31.0),
			growthRate:      constant(2.0).mix(ramp(-1.0, 8), random(0.01), unit(14).delay(25)),
			scannableFrac:   constant(1.0),
			stackBytes:      constant(8192),
			heapTargetBytes: constant(2 << 30).mix(random(512 << 20)),
			length:          50,
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
