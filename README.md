# Go GC pacer models and simulation

This repository contains a simulator for the Go GC's pacer for various 
scenarios.

Run `make` in the repository root to generate plots for all simulations, for
all scenarios.

New scenarios may be added by modifying `cmd/scenario-gen/generators` and
running `make scenarios`.
`make` will also automatically rebuild scenarios.

Models for the pacer may be found in the `simulation` package.
