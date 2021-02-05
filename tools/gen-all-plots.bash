#!/bin/bash

for file in ./data/scenarios/*; do
	case=$(echo $(basename $file) | cut -f 1 -d '.')
	for sim in $(go run ./cmd/pacer-sim -l | xargs); do
		echo "processing: $sim-$case"
		go run ./cmd/pacer-sim $sim $file | python3 $(dirname $0)/gen-plots.py $1/$sim-$case.svg
	done
done
