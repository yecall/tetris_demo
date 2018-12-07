/*
 * // Copyright (C) 2017 gyee authors
 * //
 * // This file is part of the gyee library.
 * //
 * // the gyee library is free software: you can redistribute it and/or modify
 * // it under the terms of the GNU General Public License as published by
 * // the Free Software Foundation, either version 3 of the License, or
 * // (at your option) any later version.
 * //
 * // the gyee library is distributed in the hope that it will be useful,
 * // but WITHOUT ANY WARRANTY; without even the implied warranty of
 * // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * // GNU General Public License for more details.
 * //
 * // You should have received a copy of the GNU General Public License
 * // along with the gyee library.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 */

package node

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"


	"github.com/yeeco/gyee/consensus/tetris"
	"github.com/yeeco/gyee/p2p"
	"github.com/yeeco/gyee/utils/logging"
)

const AccountNumber = 100000

type Block struct {
	Tansactions []string
	Timestamp   time.Time
}

type State struct {
	Nonce      uint64
	Balance    uint64
	PendingTxs []string
}

type Node struct {
	id         uint
	name       string //for test purpose
	members    map[string]uint
	memberAddr map[uint]string

	p2p    p2p.Service
	Tetris *tetris.Tetris
	lock   sync.RWMutex
	wg     *sync.WaitGroup
	Report chan bool
	stop   chan struct{}

	subscriberEvent *p2p.Subscriber
	subscriberTx    *p2p.Subscriber

	States     map[uint]*State
	Blockchain []Block

	running bool
}


func NewNode(id, number uint) (*Node, error) {
	members := map[string]uint{}
	memberAddr := map[uint]string{}
	for i := uint(0); i < number; i++ {
		members[strconv.Itoa(int(i))] = i
		memberAddr[i] = strconv.Itoa(int(i))
	}

	node := &Node{
		id:         id,
		name:       memberAddr[id],
		members:    members,
		memberAddr: memberAddr,
		running:    false,
	}

	p2p, err := p2p.NewInmemService()
	if err != nil {
		logging.Logger.Panic(err)
	}
	node.p2p = p2p

	tetris, err := tetris.NewTetris(nil, members, 0, memberAddr[id])
	if err != nil {
		logging.Logger.Fatal("create tetris err ", err)
	}
	node.Tetris = tetris

	node.States = make(map[uint]*State)
	for i := uint(0); i < AccountNumber; i++ {
		node.States[i] = &State{0, 100000, []string{}}
	}

	node.Blockchain = make([]Block, 0)

	node.stop = make(chan struct{})
	node.Report = make(chan bool)
	return node, nil
}

func (n *Node) Start(wg *sync.WaitGroup) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.running {
		return errors.New("already running")
	}

	n.running = true
	logging.Logger.Info("Node ", n.name, " Start...")

	n.wg = wg
	n.wg.Add(1)
	n.subscriberEvent = p2p.NewSubscriber(n, make(chan p2p.Message), p2p.MessageTypeEvent)
	n.subscriberTx = p2p.NewSubscriber(n, make(chan p2p.Message), p2p.MessageTypeTx)
	n.p2p.Register(n.subscriberEvent)
	n.p2p.Register(n.subscriberTx)
	n.p2p.Start()
	n.Tetris.Start()

	go n.loop()

	return nil
}

func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if !n.running {
		return nil
	}
	n.running = false
	for len(n.subscriberEvent.MsgChan) > 0 {
		<-n.subscriberEvent.MsgChan
	}
	for len(n.subscriberTx.MsgChan) > 0 {
		<-n.subscriberTx.MsgChan
	}

	n.p2p.UnRegister(n.subscriberEvent)
	n.p2p.UnRegister(n.subscriberTx)
	n.p2p.Stop()
	n.Tetris.Stop()
	close(n.stop)
	return nil
}

func (n *Node) BroadcastTransactions(rps uint, num uint) {
	go func(rps uint, num uint) {
		sn := make([]uint, AccountNumber)
		for i := uint(1); i < num; i++ {
			from := rand.Intn(AccountNumber)
			for from%len(n.members) != int(n.id) {
				from = rand.Intn(AccountNumber)
			}
			to := rand.Intn(AccountNumber)
			balance := rand.Intn(100)
			sn[from] = sn[from] + 1
			data := fmt.Sprintf("%6d,%d,%d,%d", sn[from], from, to, balance)
			n.p2p.BroadcastMessage(p2p.Message{p2p.MessageTypeTx, n.name, []byte(data)})
			time.Sleep(time.Second / time.Duration(rps))
		}
		time.Sleep(1 * time.Second)
		n.Stop()
	}(rps, num)

}

func (n *Node) loop() {
	//logging.Logger.Info("Node loop...")
	for {
		select {
		case <-n.stop:
			//logging.Logger.Info("Node loop end.")
			n.wg.Done()
			return
		case output := <-n.Tetris.OutputCh:
			states := n.States
			block := Block{make([]string, 0), time.Now()}
			for _, tx := range output.Tx {
				strs := strings.Split(tx, ",")
				nonce, _ := strconv.ParseUint(strings.TrimSpace(strs[0]), 10, 64)
				from, _ := strconv.ParseUint(strings.TrimSpace(strs[1]), 10, 32)
				to, _ := strconv.ParseUint(strings.TrimSpace(strs[2]), 10, 32)
				balance, _ := strconv.ParseUint(strs[3], 10, 64)

				if nonce > states[uint(from)].Nonce {
					if nonce == states[uint(from)].Nonce+1 {
						states[uint(from)].Nonce = nonce
						states[uint(from)].Balance -= balance
						states[uint(to)].Balance += balance
						block.Tansactions = append(block.Tansactions, string(tx))
						for _, ptx := range states[uint(from)].PendingTxs {
							strs := strings.Split(ptx, ",")
							nonce, _ := strconv.ParseUint(strings.TrimSpace(strs[0]), 10, 64)
							from, _ := strconv.ParseUint(strings.TrimSpace(strs[1]), 10, 32)
							to, _ := strconv.ParseUint(strings.TrimSpace(strs[2]), 10, 32)
							balance, _ := strconv.ParseUint(strs[3], 10, 64)
							if nonce == states[uint(from)].Nonce+1 {
								states[uint(from)].Nonce = nonce
								states[uint(from)].Balance -= balance
								states[uint(to)].Balance += balance
								block.Tansactions = append(block.Tansactions, string(tx))
								states[uint(from)].PendingTxs = states[uint(from)].PendingTxs[1:]
							} else {
								break
							}
						}
					} else {
						states[uint(from)].PendingTxs = append(states[uint(from)].PendingTxs, tx)
					}

				} else {
					//fmt.Println(from, " ", nonce, " < ", states[uint(from)].Nonce)
				}
			}
			n.Blockchain = append(n.Blockchain, block)
			n.States = states
			n.Report <- true
		case mevent := <-n.subscriberEvent.MsgChan:
			var event tetris.Event
			event.Unmarshal(mevent.Data)
			if len(n.Tetris.EventCh) < 100 { //这个地方有可能阻塞。。。
				n.Tetris.EventCh <- event
			} else {
				//fmt.Println("EventCh:", len(n.Tetris.EventCh))
			}

			//logging.Logger.Info("node receive ", mevent.MsgType, " ", mevent.From)
		case mtx := <-n.subscriberTx.MsgChan:
			//logging.Logger.Info("node receive ", mtx.MsgType, " ", mtx.From, " ", string(mtx.Data))
			//if len(n.Tetris.TxsCh) < 10000 { //这个地方有可能阻塞。。。
			n.Tetris.TxsCh <- string(mtx.Data)
			//} else {
			//	//fmt.Println("Txs:", len(n.Tetris.TxsCh))
			//}

		case event := <-n.Tetris.SendEventCh:
			n.p2p.BroadcastMessage(p2p.Message{p2p.MessageTypeEvent, n.name, event.Marshal()})
			n.p2p.DhtSetValue([]byte(event.Hex()), event.Marshal())
			//logging.Logger.Info("send: ", event.Body.N)
		case hex := <-n.Tetris.RequestEventCh:
			var event tetris.Event
			data, err := n.p2p.DhtGetValue([]byte(hex))
			if err != nil {
				fmt.Println("dhtGetValue error")
			}
			event.Unmarshal(data)
			go func(event tetris.Event) {
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				n.Tetris.ParentEventCh <- event
			}(event)
		}
	}
}
