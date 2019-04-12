import os
import time
import socket
import sys

hostname = socket.gethostname() 
addr = socket.gethostbyname(hostname)  

# Open clients, on ports 9000+n
os.system("rm *.log")
num_clients = int(sys.argv[1])
for i in range(num_clients):
    newpid = os.fork()
    if newpid == 0:
        port = 9000 + i
        logfile = hostname + "_" + str(port) + ".log" # logfile: IP_PORT.log
        os.system("./client " + str(9900+i) + " > " + logfile)
        break
    else:
        time.sleep(0.5)