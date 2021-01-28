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

func (s *go117) Step(gc *scenario.Cycle) Result {
	// Simulate up to when GC starts.
	//
	// 1. Figure out the goal.
	// 2. Figure out the trigger.
	// 3. Figure out the worst-case scan work.

	heapGoal := uint64(float64(s.liveBytesLast+gc.StackBytes+s.GlobalsBytes) * s.Gamma)
	if heapGoal < 4<<20 {
		heapGoal = 4 << 20
	}

	// maxScanWork = liveScannableLast + t * allocRate * scannableFrac
	// expectedScan = maxScanWork/gamma + stackBytes + globalsBytes
	// Trigger condition: t * allocRate + liveBytesLast >= heapGoal - r * expectedScan
	// => t * allocRate + liveBytesLast = heapGoal - r * ((liveScannableLast + t * allocRate * scannableFrac) / gamma + stackBytes + globalsBytes)
	// => t * allocRate + liveBytesLast = heapGoal - r * liveScannableLast / gamma - r * t * allocRate * scannableFrac / gamma - r * (stackBytes + globalsBytes)
	// => t * allocRate = heapGoal - r * liveScannableLast / gamma - r * t * allocRate * scannableFrac / gamma - r * (stackBytes + globalsBytes) - liveBytesLast
	// => t * allocRate + t * allocRate * r * scannableFrac / gamma = heapGoal - r * liveScannableLast / gamma - r * (stackBytes + globalsBytes) - liveBytesLast
	// => t * allocRate * (1 + r * scannableFrac / gamma) = heapGoal - r * liveScannableLast / gamma - r * (stackBytes + globalsBytes) - liveBytesLast
	// => t * allocRate = (heapGoal - r * (liveScannableLast / gamma + stackBytes + globalsBytes) - liveBytesLast) / (1 + r * scannableFrac / gamma)
	//
	// => extraTilTrigger = (heapGoal - r * (liveScannableLast / gamma + stackBytes + globalsBytes) - liveBytesLast) / (1 + r * scannableFrac / gamma)
	// => triggerPoint = liveBytesLast + extraTilTrigger
	var extraTilTrigger, triggerPoint uint64
	if s.gc == 0 {
		extraTilTrigger = 7 * heapGoal / 8
		triggerPoint = 7 * heapGoal / 8
	} else {
		extraTilTrigger = uint64((float64(heapGoal) - s.rValue*(float64(s.liveScannableLast)/s.Gamma+float64(gc.StackBytes)+float64(s.GlobalsBytes)) - float64(s.liveBytesLast)) /
			(1.0 + s.rValue*gc.ScannableFrac/s.Gamma))
		triggerPoint = s.liveBytesLast + extraTilTrigger
	}
	//maxHeapScanWork := s.liveScannableLast + uint64(float64(extraTilTrigger)*gc.ScannableFrac)
	//expScanWork := uint64(float64(maxHeapScanWork)/s.Gamma) + gc.StackBytes + s.GlobalsBytes

	// Simulate during-GC pacing.
	//
	// (???) 1. Figure out the assist ratio.
	// 2. Figure out the amount of scan work that will be done.
	// 3. Figure out how much of that scan work is done by assists.
	// 4. Figure out what the heap actually grows to.

	//assistRatio := (s.gamma*float64(heapGoal) - float64(triggerPoint)) / float64(maxScanWork)

	var totalScanWork uint64
	if s.gc == 0 {
		totalScanWork = s.InitialHeap + gc.StackBytes + s.GlobalsBytes
	} else {
		totalScanWork = uint64(float64(s.liveScannableLast)*gc.GrowthRate) + gc.StackBytes + s.GlobalsBytes
	}

	// Rely on the during-GC pacer to work perfectly.
	//
	// extra = (allocRate * (1-u)) / (scanRate * u) * totalScanWork
	// Set a hard goal of gamma * heapGoal.
	const u = 0.25
	peakExtra := uint64((gc.AllocRate * (1 - u)) / (gc.ScanRate * u) * float64(totalScanWork))

	peakHeap := triggerPoint + peakExtra
	assistScanWork := uint64(0)
	actualU := float64(u)
	if max := uint64(s.Gamma * float64(heapGoal)); peakHeap > max {
		peakHeap = max
		// extra = (allocRate * (1-u)) / (scanRate * u) * totalScanWork
		// => extra = (allocRate - allocRate * u) / (scanRate * u) * totalScanWork
		// => extra * scanRate * u = (allocRate - allocRate * u) * totalScanWork
		// => extra * scanRate * su = allocRate * totalScanWork - allocRate * u * totalScanWork
		// => extra * scanRate * u + allocRate * u * totalScanWork = allocRate * totalScanWork
		// => u * (extra * scanRate + allocRate * totalScanWork) = allocRate * totalScanWork
		// => u = (allocRate * totalScanWork) / (extra * scanRate + allocRate * totalScanWork)
		// if x = allocRate * totalScanWork / (extra * scanRate)
		// => u = x / (1 + x)
		x := gc.AllocRate * float64(totalScanWork) / (float64(peakHeap-triggerPoint) * gc.ScanRate)
		actualU = x / (1 + x)
		assistScanWork = uint64(float64(totalScanWork) * (actualU - u) / actualU)
	}

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

	rMeasured := float64(peakHeap-triggerPoint) / float64(totalScanWork-assistScanWork)

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
		R:             s.rValue,
		LiveBytes:     heapSurvived,
		LiveScanBytes: heapScannableSurvived,
		GCUtilization: actualU,
		TriggerPoint:  triggerPoint,
		PeakBytes:     peakHeap,
	}
}
