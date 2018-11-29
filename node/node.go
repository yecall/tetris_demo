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
	"strconv"
	"sync"
	"time"

	"fmt"
	"github.com/yeeco/gyee/consensus/tetris"
	"github.com/yeeco/gyee/p2p"
	"github.com/yeeco/gyee/utils/logging"
	"math/rand"
	"strings"
	"errors"
)

type Block struct {
	Tansactions []string
	Timestamp   time.Time
}

type State struct {
	Nonce   uint64
	Balance uint64
}

type Node struct {
	id         uint
	name       string //for test purpose
	members    map[string]uint
	memberAddr map[uint]string

	p2p    p2p.Service
	tetris *tetris.Tetris
	lock   sync.RWMutex
	wg     *sync.WaitGroup
	Report chan bool
	stop   chan struct{}

	subscriberEvent *p2p.Subscriber
	subscriberTx    *p2p.Subscriber

	States     map[string]State
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
	node.tetris = tetris

	node.States = make(map[string]State)
	for i := uint(0); i < number; i++ {
		node.States[memberAddr[i]] = State{0, 100000}
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
	n.tetris.Start()

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
	n.p2p.UnRegister(n.subscriberEvent)
	n.p2p.UnRegister(n.subscriberTx)
	n.p2p.Stop()
	n.tetris.Stop()
	//logging.Logger.Info("Node ", n.name, " Stop...")
	close(n.stop)
	return nil
}

func (n *Node) BroadcastTransactions(rps uint, num uint) {
	go func(rps uint, num uint) {
		for i := uint(0); i < num; i++ {
			to := rand.Intn(len(n.members))
			balance := rand.Intn(100)
			data := fmt.Sprintf("%6d,%s,%d,%d", i, n.name, to, balance)
			n.p2p.BroadcastMessage(p2p.Message{p2p.MessageTypeTx, n.name, []byte(data)})
			time.Sleep(time.Second / time.Duration(rps))
		}
		time.Sleep(1 * time.Second)
		n.Stop()
	}(rps, num)

}

func (n *Node) loop() {
	//n.wg.Add(1)
	//defer n.wg.Done()
	//logging.Logger.Info("Node loop...")
	for {
		select {
		case <-n.stop:
			//logging.Logger.Info("Node loop end.")
			n.wg.Done()
			return
		case output := <-n.tetris.OutputCh:
			states := n.States
			block := Block{make([]string, 0), time.Now()}
			//fmt.Println("new block")
			for _, tx := range output.Tx {
				strs := strings.Split(tx, ",")
				nonce, _ := strconv.ParseUint(strings.TrimSpace(strs[0]), 10, 64)
				from := strs[1]
				to := strs[2]
				balance, _ := strconv.ParseUint(strs[3], 10, 64)
				//fmt.Println(string(tx), " ", strs, " ", nonce, " ", from, " ", to, " ", balance)
				state := states[from]
				if nonce > state.Nonce {
					//if nonce - state.Nonce > 1 {
					//	fmt.Println(from, " ", nonce, " > ", state.Nonce)
					//}
					state.Nonce = nonce
					state.Balance -= balance
					states[from] = state
					state = states[to]
					state.Balance += balance
					states[to] = state
					block.Tansactions = append(block.Tansactions, string(tx))
				} else {
					//fmt.Println(from, " ", nonce, " < ", state.Nonce)
				}
			}
			n.Blockchain = append(n.Blockchain, block)
			n.States = states
			n.Report <- true
		case mevent := <-n.subscriberEvent.MsgChan:
			var event tetris.Event
			event.Unmarshal(mevent.Data)
			n.tetris.EventCh <- event
			//logging.Logger.Info("node receive ", mevent.MsgType, " ", mevent.From)
		case mtx := <-n.subscriberTx.MsgChan:
			//logging.Logger.Info("node receive ", mtx.MsgType, " ", mtx.From, " ", string(mtx.Data))
			n.tetris.TxsCh <- string(mtx.Data)
		case event := <-n.tetris.SendEventCh:
			n.p2p.BroadcastMessage(p2p.Message{p2p.MessageTypeEvent, n.name, event.Marshal()})
			n.p2p.DhtSetValue([]byte(event.Hex()), event.Marshal())
			//logging.Logger.Info("send: ", event.Body.N)
		case hex := <-n.tetris.RequestEventCh:
			var event tetris.Event
			data, err := n.p2p.DhtGetValue([]byte(hex))
			if err != nil {
				fmt.Println("dhtGetValue error")
			}
			event.Unmarshal(data)
			go func(event tetris.Event) {
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				n.tetris.ParentEventCh <- event
			}(event)
		}
	}
}
