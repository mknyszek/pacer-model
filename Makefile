all: scenarios clean-plots
	mkdir -p ./plots
	bash tools/gen-all-plots.bash ./plots

scenarios: clean-scenarios
	rm -f ./data/scenarios/*
	go run ./cmd/scenario-gen -o ./data/scenarios

clean: clean-plots clean-scenarios

clean-plots:
	rm -rf ./plots

clean-scenarios:
	rm -f ./data/scenarios/*
