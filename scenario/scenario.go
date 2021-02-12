package scenario

type Execution struct {
	Cycles  []Cycle `json:"cycles"`
	Globals Globals `json:"global"`
}

type Cycle struct {
	AllocRate       float64 `json:"alloc_rate"`
	ScanRate        float64 `json:"scan_rate"`
	GrowthRate      float64 `json:"growth_rate"`
	ScannableFrac   float64 `json:"scannable_frac"`
	StackBytes      uint64  `json:"stack_bytes"`
	HeapTargetBytes int64   `json:"heap_target"`
}

type Globals struct {
	Gamma        float64 `json:"gamma"`
	GlobalsBytes uint64  `json:"globals_bytes"`
	InitialHeap  uint64  `json:"init_live_heap"`
}
