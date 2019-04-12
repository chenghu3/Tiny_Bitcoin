import csv
from datetime import datetime
import numpy as np
import statistics
import matplotlib.pyplot as plt

logs = []
for j in ["01", "02", "03", "04", "05", "06","08","09","10"]:
    print("Processing machine", j)
    nodes = 11
    if (j == "10"):
        nodes = 12
    for i in range(nodes):
        print("Processing node", i)
        fname = "sp19-cs425-g10-"+j+".cs.illinois.edu_"+str(9000+i)+".log"
        logs.append([])
        with open(fname) as f:
            reader = csv.reader(f, delimiter=' ')
            for row in reader:
                if (len(row) != 0 and row[0] == "SWITCH"):
                    print(row)
                    print(fname)
                logs[i].append(row)

transactionTimes = {}
block_departure_times = {}
block_arrival_times = {}
transaction_commit_delays = {}

for logfile in logs:
    for line in logfile:
        if len(line) > 0 and line[0] == "LOG":
            tid = line[5]
            ts_float = float(line[4])
            if ts_float not in transactionTimes:
                transactionTimes[ts_float] = []
            timeObj = datetime.strptime(line[1] + " " + line[2], '%Y-%m-%d %H:%M:%S.%f')
            timestamp = datetime.fromtimestamp(ts_float)
            transactionTimes[ts_float].append((timeObj-timestamp).total_seconds())
        if len(line) > 0 and line[0] == "NEWBLOCK":
            block_id = line[3] + line[5]
            timestamp = datetime.strptime(line[1] + " " + line[2], '%Y-%m-%d %H:%M:%S.%f')
            block_departure_times[block_id] = timestamp
        if len(line) > 0 and line[0] == "RECEIVENEWBLOCK":
            block_id = line[3] + line[5]
            if block_id not in block_arrival_times:
                block_arrival_times[block_id] = []
            timestamp = datetime.strptime(line[1] + " " + line[2], '%Y-%m-%d %H:%M:%S.%f')
            block_arrival_times[block_id].append(timestamp)
        if len(line) > 0 and line[0] == "BLOCKTRANSACTION":
            ts_float = float(line[4])
            block_id = line[1] + line[2]
            block_departure_time = block_departure_times[block_id]
            timestamp = datetime.fromtimestamp(ts_float)
            transaction_commit_delays[ts_float] = (block_departure_time-timestamp).total_seconds()

block_delays = {}
for block in block_arrival_times:
    for timestamp in block_arrival_times[block]:
        if block not in block_delays:
            block_delays[block] = []
        block_delays[block].append((timestamp - block_departure_times[block]).total_seconds())

ts = list(transactionTimes.keys())
delays = list(transactionTimes.values())
max_delays = np.array([max(l) for l in delays])
min_delays = np.array([min(l) for l in delays])
median_delays = np.array([statistics.median(l) for l in delays])
mean_delays = np.array([statistics.mean(l) for l in delays])
log_counts = np.array([len(l) for l in delays])

block_ts = list(block_delays.keys())
block_delays_val = list(block_delays.values())
block_max_delays = np.array([max(l) for l in block_delays_val])
block_min_delays = np.array([min(l) for l in block_delays_val])
block_median_delays = np.array([statistics.median(l) for l in block_delays_val])
block_mean_delays = np.array([statistics.mean(l) for l in block_delays_val])
block_log_counts = np.array([len(l)+1 for l in block_delays_val]) # Include the sending block

fig = plt.figure(figsize=(20,10))
plt.errorbar(range(len(ts)), median_delays, [median_delays - min_delays, max_delays - median_delays],
             fmt='.k', ecolor='gray', lw=1)
plt.plot(log_counts)
# plt.gca().legend(("Delivered Node Count"))
plt.xlabel('Transaction ID', fontsize=12)
plt.ylabel('Time(s) and Transaction Delivery', fontsize=14)
plt.savefig("Propagation.jpg", dpi=fig.dpi)
plt.show()


fig = plt.figure(figsize=(20,10))
plt.errorbar(range(len(block_ts)), block_median_delays, [block_median_delays - block_min_delays, block_max_delays - block_median_delays],
             fmt='.k', ecolor='gray', lw=1)
plt.xlabel('Node Index', fontsize=12)
plt.ylabel('Time(s) and Node Delivery', fontsize=14)
plt.plot(block_log_counts)
plt.savefig("blockplot.jpg", dpi=fig.dpi)
plt.show()

fig = plt.figure(figsize=(20,10))
commit_delays_val = transaction_commit_delays.values()
plt.xlabel('Transaction ID', fontsize=12)
plt.ylabel('Time To Commit(s)', fontsize=14)
plt.plot(range(len(commit_delays_val)), commit_delays_val)
plt.savefig("timetocommit.jpg", dpi=fig.dpi)
plt.show()