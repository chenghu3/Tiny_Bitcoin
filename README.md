<!-- # CS425 MP2 - Cryptocurrency -->

## Group member
Jianfeng Xia(jxia11), Cheng Hu(chenghu3)

<!-- ## Environment
Go 1.11.4, Python 36

## Usage
* Log into group VMs and clone repository
* To build:
    `go build client.go`
* Run program:
    `./client PORT`  
  Alternatively, use our Python3 script `experiment.py` to start multiple clients (starting a node every 0.5s) and write stdout to logfiles.  
  Usage of Python script:  
    `python36 experiment.py [NUMBER OF CLIENTS]`
* Plotting:
    We have also included scripts to analyze performance and generate plots. To run the scripts: 
    * `cd logs`
    * `python3 propagation_plot.py`
    * `python3 bandwidth_plot.py`  
  	Note:
      1. Our plot scripts use `matplotlib` and `numpy`, we suggest to run those scripts on machine has those library.
      2. Because the log files are large(we use logs in the case of 100 nodes, 20 mesg/s), please be patient when running scripts. -->

## Thoughts on CP2:
1. Mantain a list of transactions for including transactions into block in time order, this list need to update(delete some transactions) when recieve a block.(special case same as below)
   Wrong: use set delete, poll 500 transaction when try puzzle(sort by time?)
2. Mantain a map of account and balance, update only when recieve a block. (case for drop current working chain and swith to longest chain, maybe ask block sender to send his map?)  Wrong: set balence along with block
3. Brodcast block, just use randomed gossip since it's not frequent? Big issue here, can't use StringSet for this part.
4. How to send block struct, test gob package?
5. safety (one transaction will not be mined twice): By update transactions list, but update list is O(N), try more efficent approach?
6. liveness (one transaction will be mined): Worry about the early transations that don't exits in every node, ie. old transactions for new nodes joining.
7. Discuss this post: https://piazza.com/class/jqxvctrwztu5f6?cid=499 


## TODO:
1. change introduction server to known server