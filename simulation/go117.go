package simulation

import (
	"github.com/mknyszek/pacer-model/controller"
	"github.com/mknyszek/pacer-model/scenario"
)

type go117 struct {
	scenario.Globals
	ctrl controller.Controller

	// State
	gc                      int
	liveBytesLast           uint64
	liveScannableLast       uint64
	allocBlackLast          uint64
	allocBlackScannableLast uint64
	rValue                  float64
}

const go117HeapMinimum = 2 << 20

func (s *go117) Step(gc *scenario.Cycle) Result {
	// Simulate up to when GC starts.
	//
	// 1. Figure out the goal.
	// 2. Figure out the trigger.
	// 3. Figure out the worst-case scan work.

	heapGoal := uint64(float64(s.liveBytesLast+gc.StackBytes+s.GlobalsBytes) * s.Gamma)
	if target := gc.HeapTargetBytes; target > 0 && heapGoal < uint64(target) {
		heapGoal = uint64(target)
	}
	if heapGoal < go117HeapMinimum {
		heapGoal = go117HeapMinimum
	}

	// expectedScan = liveScannableLast + stackBytes + globalsBytes
	// Trigger condition: t * allocRate + liveBytesLast >= heapGoal - r * expectedScan
	// => t * allocRate + liveBytesLast = heapGoal - r * (liveScannableLast + stackBytes + globalsBytes)
	// => t * allocRate = heapGoal - r * (liveScannableLast + stackBytes + globalsBytes) - liveBytesLast
	//
	// => extraTilTrigger = heapGoal - r * (liveScannableLast + stackBytes + globalsBytes) - liveBytesLast
	// => triggerPoint = liveBytesLast + extraTilTrigger
	var extraTilTrigger, triggerPoint uint64
	if s.gc == 0 {
		//extraTilTrigger = 7 * heapGoal / 8
		triggerPoint = 7 * heapGoal / 8
	} else {
		backwards := uint64(s.rValue*float64(s.liveScannableLast+gc.StackBytes+s.GlobalsBytes)) + s.liveBytesLast
		if backwards < heapGoal {
			extraTilTrigger = heapGoal - backwards
		}
		triggerPoint = s.liveBytesLast + extraTilTrigger
		if minTrigger := uint64(float64(heapGoal-s.liveBytesLast)*0.6) + s.liveBytesLast; triggerPoint < minTrigger {
			triggerPoint = minTrigger
			extraTilTrigger = minTrigger - s.liveBytesLast
		}
	}

	// Simulate during-GC pacing.
	assistRatio := (float64(heapGoal) - float64(triggerPoint)) / float64(s.liveScannableLast+gc.StackBytes+s.GlobalsBytes)

	var totalScanWork uint64
	if s.gc == 0 {
		totalScanWork = s.InitialHeap + gc.StackBytes + s.GlobalsBytes
	} else {
		totalScanWork = uint64(float64(s.liveScannableLast)*gc.GrowthRate) + gc.StackBytes + s.GlobalsBytes
	}

	// Rely on the during-GC pacer to work perfectly.
	//
	// Set a hard goal of gamma * heapGoal.
	const u = 0.25
	actualRatio := (gc.AllocRate * (1 - u)) / (gc.ScanRate * u)
	actualU := float64(u)
	if actualRatio > assistRatio {
		actualRatio = assistRatio
		// ratio = (allocRate * (1-u)) / (scanRate * u)
		// => ratio = (allocRate - allocRate * u) / (scanRate * u)
		// => ratio * scanRate * u = allocRate - allocRate * u
		// => ratio * scanRate * u + allocRate * u = allocRate
		// => u * (ratio * scanRate + allocRate) = allocRate
		// => u = allocRate / (ratio * scanRate + allocRate)
		// if x = allocRate / (ratio * scanRate)
		// => u = x / (1 + x)
		x := gc.AllocRate / (actualRatio * gc.ScanRate)
		actualU = x / (1 + x)
	}
	peakExtra := uint64(actualRatio * float64(totalScanWork))
	peakHeap := triggerPoint + peakExtra

	// Simulate GC feedback loop.
	//
	// 1. Figure out how much survived this GC.
	// 2. Use that and other values computed earlier to determine
	//    what r was and our setpoint for r.
	// 3. Run a step of the PI controller.
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

	rMeasured := float64(peakHeap-triggerPoint) / float64(totalScanWork) * ((1 - u) / (1 - actualU)) / (u / actualU)

	thisR := s.rValue
	s.rValue += s.ctrl.Next(s.rValue, rMeasured)
	if s.rValue < 0.05 {
		s.rValue = 0.05
	} else if s.rValue > s.Gamma-0.05 {
		s.rValue = s.Gamma - 0.05
	}
	s.liveBytesLast = heapSurvived
	s.liveScannableLast = heapScannableSurvived
	s.allocBlackLast = heapAllocBlack
	s.allocBlackScannableLast = heapAllocBlackScannable

	// Final bookkeeping.
	s.gc++

	// Return result.
	return Result{
		R:                   thisR,
		LiveBytes:           heapSurvived,
		LiveScanBytes:       heapScannableSurvived,
		GoalBytes:           heapGoal,
		ActualGCUtilization: actualU,
		TargetGCUtilization: u,
		TriggerPoint:        triggerPoint,
		PeakBytes:           peakHeap,
	}
}
