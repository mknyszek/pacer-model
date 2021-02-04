package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mknyszek/pacer-model/controller"
	"github.com/mknyszek/pacer-model/scenario"
	"github.com/mknyszek/pacer-model/simulation"
)

var (
	genJSONFlag    *bool   = flag.Bool("json", false, "generate a JSON file instead of a CSV")
	ctrlConfigFlag *string = flag.String("controller-config", "", "file containing JSON controller configuration (optional, default parameters used otherwise)")
	listFlag       *bool   = flag.Bool("l", false, "list available pacers")
)

func run() error {
	flag.Parse()

	if *listFlag {
		fmt.Println(strings.Join(simulation.Simulators(), "\n"))
		return nil
	}

	if flag.NArg() != 2 {
		return fmt.Errorf("expected 2 arguments: pacer type and scenario file")
	}

	// Parse scenario.
	scnData, err := ioutil.ReadFile(flag.Arg(1))
	if err != nil {
		return err
	}
	var scn scenario.Execution
	if err := json.Unmarshal(scnData, &scn); err != nil {
		return fmt.Errorf("unmarshalling scenario data: %v", err)
	}

	// Parse controller configuration.
	var ctrl controller.Controller
	if *ctrlConfigFlag != "" {
		ctrlData, err := ioutil.ReadFile(*ctrlConfigFlag)
		if err != nil {
			return err
		}
		var ctrlCfg controller.PIConfig
		if err := json.Unmarshal(ctrlData, &ctrlCfg); err != nil {
			return fmt.Errorf("unmarshalling controller config: %v", err)
		}
		ctrl = controller.NewPI(&ctrlCfg)
	}

	// Pick a simulator and inject a controller.
	s, err := simulation.NewSimulator(flag.Arg(0), scn.Globals, ctrl)
	if err != nil {
		return err
	}

	// Compute results.
	var r []simulation.Result
	for i := range scn.Cycles {
		r = append(r, s.Step(&scn.Cycles[i]))
	}

	// Write output.
	if *genJSONFlag {
		results, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshalling results: %v", err)
		}
		fmt.Println(string(results))
	} else {
		printCSV(&scn, r)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printCSV(ex *scenario.Execution, r []simulation.Result) {
	fmt.Println("Allocation Rate,Growth Rate,Scan Rate,Scannable Rate,Stack Bytes,R,Live Bytes,Scannable Live Bytes,Goal,Actual Utilization,Target Utilization,Trigger,Peak")
	c := ex.Cycles
	for i := range r {
		fmt.Printf("%f,%f,%f,%f,%d,%f,%d,%d,%d,%f,%f,%d,%d\n",
			c[i].AllocRate,
			c[i].GrowthRate,
			c[i].ScanRate,
			c[i].ScannableFrac,
			c[i].StackBytes,
			r[i].R,
			r[i].LiveBytes,
			r[i].LiveScanBytes,
			r[i].GoalBytes,
			r[i].ActualGCUtilization,
			r[i].TargetGCUtilization,
			r[i].TriggerPoint,
			r[i].PeakBytes,
		)
	}
}
