import numpy as np
import matplotlib.pyplot as plt
import sys
import math

if len(sys.argv) == 1:
    outFile = None
elif len(sys.argv) == 2:
    outFile = sys.argv[1]
else:
    print(f"usage: {sys.argv[0]} [output file]")
    print(f"input and output handled with stdin and stdout")
    exit(1)

data = np.genfromtxt(sys.stdin, delimiter=',', names=True)
t = np.linspace(1, len(data), num=len(data))

def bytesToUnit(b):
    bp = b
    order = 0
    for order, unit in enumerate(['B', 'KiB', 'MiB', 'GiB']):
        if int(bp) >> 10*(order+1) == 0:
            return bp / float(1<<(10*order)), unit
    return bp / float(1<<40), 'TiB' 

GOGC = int((data['Gamma'][0] - 1) * 100)
globData, globUnit = bytesToUnit(data['Globals_Bytes'][0])

def doLegend(axs, handles):
    axs.legend(handles=handles, loc='lower left', \
        bbox_to_anchor=(0, 1.02, 1.0, 0.2), \
        ncol=int(math.sqrt(len(handles))), fancybox=True, shadow=True)

def bytesPlot(axs):
    goalPlot, = axs.plot(t, data['Goal'] / float(1<<20), label='Heap goal')
    triggerPlot, = axs.plot(t, data['Trigger'] / float(1<<20), label='GC trigger')
    peakPlot, = axs.plot(t, data['Peak'] / float(1<<20), label='Peak heap')
    stackPlot, = axs.plot(t, data['Stack_Bytes'] / float(1<<20), label='Stack')
    livePlot, = axs.plot(t, data['Live_Bytes'] / float(1<<20), label='Live')
    doLegend(axs, handles=[goalPlot, triggerPlot, peakPlot, stackPlot, livePlot])
    axs.set_ylabel('MiB')
    axs.set_xlim(t[0], t[-1])
    axs.grid(True)

def rPlot(axs):
    asr = data['Allocation_Rate']*(1-data['Target_Utilization']) / (data['Scan_Rate']*data['Target_Utilization'])
    rPlot, = axs.plot(t, data['R'], label='R value')
    asrPlot, = axs.plot(t, asr, label='Alloc/Scan ratio')
    doLegend(axs, handles=[asrPlot, rPlot])
    axs.set_xlim(t[0], t[-1])
    axs.set_ylim(0.0, data['Gamma'][0])
    axs.grid(True)

def uPlot(axs):
    uaPlot, = axs.plot(t, data['Actual_Utilization'], label='Actual GC CPU Util')
    utPlot, = axs.plot(t, data['Target_Utilization'], label='Target GC CPU Util')
    doLegend(axs, handles=[uaPlot, utPlot])
    axs.set_xlim(t[0], t[-1])
    axs.set_ylim(0.0, 1.0)
    axs.grid(True)

def oPlot(axs):
    overshootPlot, = axs.plot(t, ((data['Peak'] / data['Goal'])-1)*100, label='Heap overshoot')
    doLegend(axs, handles=[overshootPlot])
    axs.set_ylabel('Percent')
    axs.set_xlim(t[0], t[-1])
    axs.grid(True)

fig, axs = plt.subplots(2, 2, figsize=(8, 5.5))
fig.suptitle(f"GOGC={GOGC}, Globals={globData} {globUnit}")
fig.text(0.5, 0.04, 'GC cycle', ha='center')

bytesPlot(axs[0][0])
oPlot(axs[1][0])
uPlot(axs[0][1])
rPlot(axs[1][1])

fig.tight_layout(rect=(0, 0.05, 1, 1))

if outFile is None:
    plt.show()
else:
    plt.savefig(outFile, dpi=144)
