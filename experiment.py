import os
import time
import socket

hostname = socket.gethostname() 
addr = socket.gethostbyname(hostname)  

# Open clients, on ports 9000+n
for i in range(25):
    newpid = os.fork()
    if newpid == 0:
        port = 9000 + i
        logfile = hostname + "_" + str(port) + ".log" # logfile: IP_PORT.log
        os.system("./client " + str(9000+i) + " > " + logfile)
        break
    else:
        time.sleep(0.5)