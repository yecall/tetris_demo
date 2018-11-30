# tetris_demo
Tetris Demo

This is a demo for Tetris algorithm. 

The demo simulate N nodes on a single machine, every node has an account with initial balance of 100000, and every node issue transactions to transfer random tokens to randoem other nodes. 

Every node maintain a mini blockchain with account state and blocks generated according the consensus output of Tetris.

At the end of the demo, we can see account state of every node are consistent at the same block height.


##Installation

go get -u github.com/yeeco/tetris_demo

##Usage
```
cd tetris_demo
go run main.go

//Several parameters can be set:  
-n=10       //numbers of nodes to simulate, default 10  
-tx=20000   //numbers of transactions every node issue, default 30000  
-c=3        //numbers of nodes will crash during the test at random time, default 0. 
            //If c>n/3, then remain nodes can not reach consensus.
```

##Example
```
xujiajundeiMac:yeeco xujiajun$ cd tetris_demo
xujiajundeiMac:tetris_demo xujiajun$ go run main.go
INFO[2018-11-30T10:58:25+08:00] Node 0 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 1 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 2 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 3 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 4 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 5 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 6 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 7 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 8 Start...                              
INFO[2018-11-30T10:58:25+08:00] Node 9 Start...                              

              node0     node1     node2     node3     node4     node5     node6     node7     node8     node9     
Height(Txs):112(1919) 112(1919) 112(1919) 112(1919) 112(1919) 112(1919) 112(1919) 112(1919) 112(1919) 112(1919) 

                      balance0  balance1  balance2  balance3  balance4  balance5  balance6  balance7  balance8  balance9  
Node:0  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:1  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:2  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:3  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:4  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:5  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:6  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:7  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:8  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   
Node:9  Height:112    111803    113734    104776     96668    101320     97957     85990    111273     92726     83753   

Txs confirmed: 256553  Time: 13.433156047  Tps: 19098 
```
Height(Txs) show how many txs have been packed into a block at specific height.  
We can see at the same height of 112, every nodes's state of accounts balance are consistent.

##Comments
1. Tetris algorithm here is incomplete and just for demo purpose. The full function Tetris source code will open sourced after mainnet released.

2. The demo mainly show the performance of consensus computing, we omitted all the signature/verification of events and txs.

3. We simulated the p2p network with 50ms message delay and 10% message lost.

 
