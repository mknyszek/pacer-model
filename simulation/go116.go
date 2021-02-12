package simulation

import (
	"github.com/mknyszek/pacer-model/scenario"
)

type go116 struct {
	scenario.Globals

	// State
	gc                      int
	liveBytesLast           uint64
	liveScannableLast       uint64
	allocBlackLast          uint64
	allocBlackScannableLast uint64
	triggerRatioRaw         float64
	triggerRatio            float64
}

const go116HeapMinimum = 4 << 20

func (s *go116) Step(gc *scenario.Cycle) Result {
	// Simulate up to when GC starts.
	//
	// 1. Figure out the goal.
	// 2. Figure out the trigger.
	// 3. Figure out the worst-case scan work.

	heapGoal := uint64(float64(s.liveBytesLast) * s.Gamma)
	if target := gc.HeapTargetBytes; target > 0 && heapGoal < uint64(target) {
		heapGoal = uint64(target)
	}
	if heapGoal < go116HeapMinimum {
		heapGoal = go116HeapMinimum
	}
	if s.gc == 0 {
		s.triggerRatioRaw = 7.0 / 8.0
		s.triggerRatio = 7.0 / 8.0
	}
	triggerPoint := uint64(float64(s.liveBytesLast) * (1 + s.triggerRatio))
	maxScanWork := s.liveScannableLast + uint64(float64(triggerPoint-s.liveBytesLast)*gc.ScannableFrac)
	expScanWork := uint64(float64(maxScanWork) / s.Gamma)
	dummyLiveLast := uint64(float64(heapGoal) / (1 + s.triggerRatio))
	if s.gc == 0 {
		triggerPoint = heapGoal
		heapGoal = uint64(float64(dummyLiveLast) * s.Gamma)
		maxScanWork = dummyLiveLast + uint64(float64(triggerPoint-dummyLiveLast)*gc.ScannableFrac)
		expScanWork = uint64(float64(maxScanWork) / s.Gamma)
	}

	// Simulate during-GC pacing.
	//
	// (???) 1. Figure out the assist ratio.
	// 2. Figure out the amount of scan work that will be done.
	// 3. Figure out how much of that scan work is done by assists.
	// 4. Figure out what the heap actually grows to.

	var totalScanWork uint64
	if s.gc == 0 {
		totalScanWork = s.InitialHeap + gc.StackBytes + s.GlobalsBytes
	} else {
		totalScanWork = uint64(float64(s.liveScannableLast)*gc.GrowthRate) + gc.StackBytes + s.GlobalsBytes
	}

	// Rely on the during-GC pacer to work perfectly.
	// Target utilization is 25% + 5% for assists.
	//
	// extra = (allocRate * (1-u)) / (scanRate * u) * totalScanWork
	//
	// Set a hard goal of 1.1 * heapGoal.
	// If there's more scan work than expected, or if we exceed the heap
	// goal at any point, pace for the worst case.
	const uTargetDedicated = 0.25
	const uTargetAssist = 0.05
	const uTarget = uTargetDedicated + uTargetAssist

	// This gets complicated. allocated memory counts both toward work done
	// AND how much runway we have.
	assistRatioRelaxed := float64(heapGoal-triggerPoint) / float64(expScanWork)

	// (allocRate * (1-u)) / (scanRate * u) = assistRatio
	// allocRate - allocRate * u = scanRate * u * assistRatio
	// allocRate = allocRate * u + scanRate * u * assistRatio
	// allocRate = u * (allocRate + scanRate * assistRatio)
	// u = allocRate / (allocRate + scanRate * assistRatio)

	uExp := gc.AllocRate / (gc.AllocRate + gc.ScanRate*assistRatioRelaxed)
	if uExp < uTargetDedicated {
		uExp = uTargetDedicated
	}
	r := (gc.AllocRate * (1 - uExp)) / (gc.ScanRate * uExp)
	uActual := uExp
	hardHeapGoal := uint64(1.1 * float64(heapGoal))
	var peakHeap uint64
	if expScanWork >= totalScanWork {
		peakExtra := uint64(r * float64(totalScanWork))
		peakHeap = triggerPoint + peakExtra
		if peakHeap > heapGoal {
			scanWorkDone := uint64(float64(heapGoal-triggerPoint) / r)
			estScanWorkLeft := maxScanWork - scanWorkDone
			assistRatioPanicked := float64(hardHeapGoal-heapGoal) / float64(estScanWorkLeft)
			uWorst := gc.AllocRate / (gc.AllocRate + gc.ScanRate*assistRatioPanicked)
			if uWorst < uTargetDedicated {
				uWorst = uTargetDedicated
			}
			scanWorkLeft := totalScanWork - scanWorkDone
			// uActual * totalScanWork = uExp * expScanWork + uWorst * scanWorkLeftAtGoal
			uActual = (uExp*float64(scanWorkDone) + uWorst*float64(scanWorkLeft)) / float64(totalScanWork)
			peakHeap = hardHeapGoal
		}
	} else {
		peakExtra := uint64(r * float64(expScanWork))
		peakHeap = triggerPoint + peakExtra
		if peakHeap > heapGoal {
			scanWorkDone := uint64(float64(heapGoal-triggerPoint) / r)
			estScanWorkLeft := maxScanWork - scanWorkDone
			assistRatioPanicked := float64(hardHeapGoal-heapGoal) / float64(estScanWorkLeft)
			uWorst := gc.AllocRate / (gc.AllocRate + gc.ScanRate*assistRatioPanicked)
			if uWorst < uTargetDedicated {
				uWorst = uTargetDedicated
			}
			scanWorkLeft := totalScanWork - scanWorkDone
			uActual = (uExp*float64(scanWorkDone) + uWorst*float64(scanWorkLeft)) / float64(totalScanWork)
			peakHeap = hardHeapGoal
		} else {
			scanWorkDone := expScanWork
			estScanWorkLeft := maxScanWork - scanWorkDone
			assistRatioPanicked := float64(hardHeapGoal-peakHeap) / float64(estScanWorkLeft)
			uWorst := gc.AllocRate / (gc.AllocRate + gc.ScanRate*assistRatioPanicked)
			if uWorst < uTargetDedicated {
				uWorst = uTargetDedicated
			}
			scanWorkLeft := totalScanWork - scanWorkDone
			extra := uint64((gc.AllocRate * (1 - uWorst)) / (gc.ScanRate * uWorst) * float64(scanWorkLeft))
			uActual = (uExp*float64(scanWorkDone) + uWorst*float64(scanWorkLeft)) / float64(totalScanWork)
			peakHeap += extra
		}
	}

	// Simulate GC feedback loop.
	//
	// 1. Figure out how much survived this GC.
	// 2. Use that and other values computed earlier to determine
	//    what r was and our setpoint for r.
	// 3. Run a step of the P controller.
	// 4. Feed how much data survived to the next cycle.

	heapAllocBlack := peakHeap - triggerPoint
	heapAllocBlackScannable := uint64(float64(heapAllocBlack) * gc.ScannableFrac)

	var heapSurvived, heapScannableSurvived uint64
	if s.gc == 0 {
		heapSurvived = s.InitialHeap + heapAllocBlack
		heapScannableSurvived = uint64(float64(heapSurvived) * gc.ScannableFrac)
	} else {
		heapSurvived = uint64(float64(s.liveBytesLast-s.allocBlackLast)*gc.GrowthRate) + heapAllocBlack
		heapScannableSurvived = uint64(float64(s.liveScannableLast-s.allocBlackScannableLast)*gc.GrowthRate) + heapAllocBlackScannable
	}
	// Back out an "alloc/scan" ratio given the trigger ratio and the
	// work done last cycle.
	rValue := float64(heapGoal-triggerPoint) / float64(expScanWork)

	nextHeapGoal := uint64(float64(s.liveBytesLast) * s.Gamma)
	nextGammaGoal := true
	if target := gc.HeapTargetBytes; target > 0 && nextHeapGoal < uint64(target) {
		nextHeapGoal = uint64(target)
		nextGammaGoal = false
	}
	if nextHeapGoal < go116HeapMinimum {
		nextHeapGoal = go116HeapMinimum
		nextGammaGoal = false
	}

	goalGrowthRatio := float64(heapGoal-s.liveBytesLast) / float64(s.liveBytesLast)
	if s.gc == 0 {
		goalGrowthRatio = float64(heapGoal-dummyLiveLast) / float64(dummyLiveLast)
	}
	var actualGrowthRatio float64
	if s.gc == 0 {
		actualGrowthRatio = float64(peakHeap)/float64(dummyLiveLast) - 1
	} else {
		actualGrowthRatio = float64(peakHeap)/float64(s.liveBytesLast) - 1
	}
	delta := 0.5 * (goalGrowthRatio - s.triggerRatioRaw - uActual/uTarget*(actualGrowthRatio-s.triggerRatioRaw))
	s.triggerRatioRaw += delta
	s.triggerRatio = s.triggerRatioRaw
	if s.triggerRatio < 0.6*(s.Gamma-1) {
		s.triggerRatio = 0.6 * (s.Gamma - 1)
	} else if nextGammaGoal && s.triggerRatio > 0.95*(s.Gamma-1) {
		s.triggerRatio = 0.95 * (s.Gamma - 1)
	} else if nextGoalGR := float64(nextHeapGoal)/float64(heapSurvived) - 1; !nextGammaGoal && s.triggerRatio > 0.95*nextGoalGR {
		s.triggerRatio = 0.95 * nextGoalGR
	}
	s.liveBytesLast = heapSurvived
	s.liveScannableLast = heapScannableSurvived
	s.allocBlackLast = heapAllocBlack
	s.allocBlackScannableLast = heapAllocBlackScannable

	// Final bookkeeping.
	s.gc++

	// Return result.
	return Result{
		R:                   rValue,
		LiveBytes:           heapSurvived,
		LiveScanBytes:       heapScannableSurvived,
		GoalBytes:           heapGoal,
		ActualGCUtilization: uActual,
		TargetGCUtilization: uTarget,
		TriggerPoint:        triggerPoint,
		PeakBytes:           peakHeap,
	}
}
