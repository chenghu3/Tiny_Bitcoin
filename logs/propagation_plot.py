import csv
from datetime import datetime
import numpy as np
import statistics
import matplotlib.pyplot as plt

logs = []
for j in ["01", "02", "03", "04", "05", "06","08","09","10"]:
    nodes = 11
    if (j == "10"):
        nodes = 12
    for i in range(nodes):
        fname = "sp19-cs425-g10-"+j+".cs.illinois.edu_"+str(9000+i)+".log"
        logs.append([])
        with open(fname) as f:
            reader = csv.reader(f, delimiter=' ')
            for row in reader:
                if (row[0] == "LOG"):
                    logs[i].append(row)

transactionTimes = {}

for logfile in logs:
    for line in logfile:
        if line[0] == "LOG":
            tid = line[5]
            ts_float = float(line[4])
            if ts_float not in transactionTimes:
                transactionTimes[ts_float] = []
            timeObj = datetime.strptime(line[1] + " " + line[2], '%Y-%m-%d %H:%M:%S.%f')
            timestamp = datetime.fromtimestamp(ts_float)
            transactionTimes[ts_float].append((timeObj-timestamp).total_seconds())

t1 = "01:08:29.479544"
time1 = datetime.strptime(t1, '%H:%M:%S.%f')

ts = list(transactionTimes.keys())
delays = list(transactionTimes.values())
max_delays = np.array([max(l) for l in delays])
min_delays = np.array([min(l) for l in delays])
median_delays = np.array([statistics.median(l) for l in delays])
mean_delays = np.array([statistics.mean(l) for l in delays])
log_counts = np.array([len(l) for l in delays])

fig = plt.figure(figsize=(20,10))
plt.errorbar(range(len(ts)), median_delays, [median_delays - min_delays, max_delays - median_delays],
             fmt='.k', ecolor='gray', lw=1)
plt.plot(log_counts)
plt.gca().legend(("Delivered Node count"))
plt.xlabel('Transaction ID', fontsize=12)
plt.ylabel('Time(s) and transaction delivery', fontsize=14)
plt.savefig("Propagation.jpg", dpi=fig.dpi)
plt.show()
