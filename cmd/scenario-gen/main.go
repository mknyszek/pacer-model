package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mknyszek/pacer-model/scenario"
)

var (
	outputFlag = flag.String("o", ".", "where to output scenarios")
	filterFlag = flag.String("filter", "", "filter scenarios by name")
	listFlag   = flag.Bool("l", false, "list available scenarios")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	genNames := scenario.Generators()
	if *listFlag {
		fmt.Println(strings.Join(genNames, "\n"))
		return nil
	}
	if *filterFlag != "" {
		r, err := regexp.Compile(*filterFlag)
		if err != nil {
			return fmt.Errorf("compiling filter regexp: %v", err)
		}
		fNames := make([]string, 0, len(genNames))
		for _, name := range genNames {
			if r.MatchString(name) {
				fNames = append(fNames, name)
			}
		}
		genNames = fNames
	}
	for _, name := range genNames {
		result, err := scenario.Generate(name)
		if err != nil {
			// Internal error.
			panic(err)
		}
		path := filepath.Join(*outputFlag, fmt.Sprintf("%s.json", name))
		if err := writeScenario(result, path); err != nil {
			return fmt.Errorf("writing scenario to %q: %v", path, err)
		}
	}
	return nil
}

func writeScenario(e scenario.Execution, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	if err := enc.Encode(&e); err != nil {
		return err
	}
	return nil
}
