import numpy as np
import matplotlib.pyplot as plt
import sys

if len(sys.argv) < 2:
    f = sys.stdin
else:
    f = open(sys.argv[1], 'r')

data = np.genfromtxt(f, delimiter=',', names=True)
t = np.linspace(1, len(data), num=len(data))

fig, axs = plt.subplots(3, 1)

goalPlot, = axs[0].plot(t, data['Goal'] / float(1<<20), label='Heap goal')
triggerPlot, = axs[0].plot(t, data['Trigger'] / float(1<<20), label='GC trigger')
peakPlot, = axs[0].plot(t, data['Peak'] / float(1<<20), label='Peak heap')
stackPlot, = axs[0].plot(t, data['Stack_Bytes'] / float(1<<20), label='Stack')
axs[0].legend(handles=[goalPlot, triggerPlot, peakPlot, stackPlot], \
    loc='center left', bbox_to_anchor=(1.0, 0.5), ncol=1, fancybox=True, shadow=True)
axs[0].set_ylabel('MiB')
axs[0].set_xlim(t[0], t[-1])
axs[0].grid(True)

asr = data['Allocation_Rate']*(1-data['Target_Utilization']) / (data['Scan_Rate']*data['Target_Utilization'])
rPlot, = axs[1].plot(t, data['R'], label='R value')
asrPlot, = axs[1].plot(t, asr, label='Alloc/Scan ratio')
axs[1].legend(handles=[asrPlot, rPlot], \
    loc='center left', bbox_to_anchor=(1.0, 0.5), ncol=1, fancybox=True, shadow=True)
axs[1].set_xlim(t[0], t[-1])
axs[1].grid(True)

uaPlot, = axs[2].plot(t, data['Actual_Utilization'], label='Actual GC CPU Util')
utPlot, = axs[2].plot(t, data['Target_Utilization'], label='Target GC CPU Util')
axs[2].legend(handles=[uaPlot, utPlot], \
    loc='center left', bbox_to_anchor=(1.0, 0.5), ncol=1, fancybox=True, shadow=True)
axs[2].set_xlabel('GC cycle')
axs[2].set_xlim(t[0], t[-1])
axs[2].grid(True)

fig.tight_layout()
plt.show()
