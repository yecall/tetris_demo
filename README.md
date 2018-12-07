# tetris_demo
Tetris Demo

This is a demo for Tetris algorithm. 

The demo simulate N nodes on a single machine, and 100000 accounts with initial balance of 100000, and every node issue transactions to transfer random tokens from random account to random other account. 

Every node maintain a mini blockchain with account states and blocks generated according the consensus output of Tetris.

At the end of the demo, we can see account state of every node are consistent at the same block height.


## Installation

go get -u github.com/yeeco/tetris_demo

## Usage
```
cd tetris_demo
go run main.go

//Several parameters can be set:  
-n=10       //numbers of nodes to simulate, default 10  
-tx=20000   //numbers of transactions every node issue, default 30000  
-c=3        //numbers of nodes will crash during the test at random time, default 0. 
            //If c>n/3, then remain nodes can not reach consensus.
```

## Example
```
xujiajundeiMac:tetris_demo xujiajun$ go run main.go
INFO[2018-12-07T12:34:27+08:00] Node 0 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 1 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 2 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 3 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 4 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 5 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 6 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 7 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 8 Start...                              
INFO[2018-12-07T12:34:27+08:00] Node 9 Start...                              

              node0     node1     node2     node3     node4     node5     node6     node7     node8     node9     
Height(Txs):191(2297) 191(2297) 191(2297) 191(2297) 191(2297) 191(2297) 191(2297) 191(2297) 191(2297) 191(2297) 

                      Acc71074  Acc29936  Acc89077  Acc61070  Acc26256  Acc33325  Acc85992  Acc90046  Acc84189  Acc4213   
Node:0  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:1  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:2  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:3  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:4  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:5  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:6  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:7  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:8  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   
Node:9  Height:191     99885     99822     99959     99989     99989    100004    100143    100235     99983    100064   

Txs confirmed: 471630  Time: 28.998032217  Tps: 16264 
```
Height(Txs) show how many txs have been packed into a block at specific height.  
We can see at the same height of 191, every randomly selected account's state of balance are consistent.

## Comments
1. Tetris algorithm here is incomplete and just for demo purpose. The full function Tetris source code will be open sourced after mainnet released.

2. The demo mainly show the performance of consensus computing, we omitted all the signature/verification of events and txs.

3. We simulated the p2p network with 100ms message delay and 10% message lost.

 
