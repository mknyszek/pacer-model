package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

type Cycle struct {
	AllocRate     float64 `json:"alloc_rate"`
	ScanRate      float64 `json:"scan_rate"`
	GrowthRate    float64 `json:"growth_rate"`
	ScannableFrac float64 `json:"scannable_frac"`
	StackBytes    uint64  `json:"stack_bytes"`
}

type Scenario struct {
	Cycles  []Cycle         `json:"cycles"`
	Globals ScenarioGlobals `json:"global"`
}
type ScenarioGlobals struct {
	Gamma        float64 `json:"gamma"`
	GlobalsBytes uint64  `json:"globals_bytes"`
	InitialHeap  uint64  `json:"init_live_heap"`
}

type Result struct {
	R             float64 `json:"r"`
	LiveBytes     uint64  `json:"live"`
	LiveScanBytes uint64  `json:"scan"`
	GCUtilization float64 `json:"u"`
	TriggerPoint  uint64  `json:"trigger"`
	PeakBytes     uint64  `json:"peak"`
}

type Simulator interface {
	Step(*Cycle) Result
}

func printCSV(c []Cycle, r []Result) {
	fmt.Println("Allocation Rate,Survival Rate,Scan Rate,Scannable Rate,Stack Bytes,R,Live Bytes,Scannable Live Bytes,Utilization,Trigger,Peak")
	for i := range r {
		fmt.Printf("%f,%f,%f,%f,%d,%f,%d,%d,%f,%d,%d\n",
			c[i].AllocRate,
			c[i].GrowthRate,
			c[i].ScanRate,
			c[i].ScannableFrac,
			c[i].StackBytes,
			r[i].R,
			r[i].LiveBytes,
			r[i].LiveScanBytes,
			r[i].GCUtilization,
			r[i].TriggerPoint,
			r[i].PeakBytes)
	}
}

var genJSONFlag *bool = flag.Bool("json", false, "generate a JSON file instead of a CSV")

func run() error {
	flag.Parse()

	if flag.NArg() != 3 {
		return fmt.Errorf("expected 3 arguments: pacer type, scenario file, and controller config")
	}

	scnData, err := ioutil.ReadFile(flag.Arg(1))
	if err != nil {
		return err
	}
	var scn Scenario
	if err := json.Unmarshal(scnData, &scn); err != nil {
		return fmt.Errorf("unmarshalling scenario data: %v", err)
	}

	ctrlData, err := ioutil.ReadFile(flag.Arg(2))
	if err != nil {
		return err
	}
	var ctrlCfg ControllerConfig
	if err := json.Unmarshal(ctrlData, &ctrlCfg); err != nil {
		return fmt.Errorf("unmarshalling controller config: %v", err)
	}

	c := NewController(ctrlCfg)
	var s Simulator
	switch flag.Arg(0) {
	case "old":
		s = &pacerOldSim{
			ScenarioGlobals: scn.Globals,
		}
	case "new":
		s = &pacerNewSim{
			ScenarioGlobals: scn.Globals,
			Controller:      c,
		}
	default:
		return fmt.Errorf("pacer type must be one of: new old")
	}

	var r []Result
	for i := range scn.Cycles {
		r = append(r, s.Step(&scn.Cycles[i]))
	}

	if *genJSONFlag {
		results, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshalling results: %v", err)
		}
		fmt.Println(string(results))
	} else {
		printCSV(scn.Cycles, r)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
