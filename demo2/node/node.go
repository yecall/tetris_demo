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
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"encoding/hex"
	"github.com/yeeco/gyee/common"
	"github.com/yeeco/gyee/consensus/tetris2"
	"github.com/yeeco/gyee/crypto"
	"github.com/yeeco/gyee/p2p"
	"github.com/yeeco/gyee/utils/logging"
)

const AccountNumber = 10000

type Block struct {
	Tansactions []string
	Timestamp   time.Time
}

type State struct {
	Nonce      uint64
	Balance    uint64
	PendingTxs []common.Hash
}

type Node struct {
	id      uint
	name    string //for test purpose
	members []string

	p2p    p2p.Service
	Tetris *tetris2.Tetris
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
	members := make([]string, number)

	for i := uint(0); i < number; i++ {
		s := strconv.Itoa(int(i))
		if i<10 {
			s = "0" + s
		}
		members[i] = hex.EncodeToString([]byte(s))
		for j := 0; j < 5; j++ {
			members[i] = members[i] + members[i]
		}
	}

	node := &Node{
		id:      id,
		name:    members[id],
		members: members,
		running: false,
	}

	p2p, err := p2p.NewInmemService()

	if err != nil {
		logging.Logger.Panic(err)
	}
	node.p2p = p2p

	tetris, err := tetris2.NewTetris(node, node.name, members, 0)
	if err != nil {
		logging.Logger.Fatal("create tetris err ", err)
	}
	node.Tetris = tetris

	node.States = make(map[uint]*State)
	for i := uint(0); i < AccountNumber; i++ {
		node.States[i] = &State{0, 100000, []common.Hash{}}
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

	n.p2p.Start()
	n.subscriberEvent = p2p.NewSubscriber(n, make(chan p2p.Message), p2p.MessageTypeEvent)
	n.subscriberTx = p2p.NewSubscriber(n, make(chan p2p.Message), p2p.MessageTypeTx)
	n.p2p.Register(n.subscriberEvent)
	n.p2p.Register(n.subscriberTx)

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

func (n *Node) Running() bool {
	n.lock.Lock()
	defer n.lock.Unlock()

	return n.running
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

			txData := make([]byte, 32)
			binary.BigEndian.PutUint64(txData[0:8], uint64(sn[from]))
			binary.BigEndian.PutUint64(txData[8:16], uint64(from))
			binary.BigEndian.PutUint64(txData[16:24], uint64(to))
			binary.BigEndian.PutUint64(txData[24:32], uint64(balance))

			n.p2p.BroadcastMessage(p2p.Message{p2p.MessageTypeTx, n.name, nil, txData})
			time.Sleep(time.Second / time.Duration(2*rps))
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
				nonce := binary.BigEndian.Uint64(tx[0:8])
				from := binary.BigEndian.Uint64(tx[8:16])
				to := binary.BigEndian.Uint64(tx[16:24])
				balance := binary.BigEndian.Uint64(tx[24:32])

				if nonce > states[uint(from)].Nonce {
					if nonce == states[uint(from)].Nonce+1 {
						states[uint(from)].Nonce = nonce
						states[uint(from)].Balance -= balance
						states[uint(to)].Balance += balance
						block.Tansactions = append(block.Tansactions, string(tx[:]))
						for _, ptx := range states[uint(from)].PendingTxs {
							nonce := binary.BigEndian.Uint64(ptx[0:8])
							from := binary.BigEndian.Uint64(ptx[8:16])
							to := binary.BigEndian.Uint64(ptx[16:24])
							balance := binary.BigEndian.Uint64(ptx[24:32])
							if nonce == states[uint(from)].Nonce+1 {
								states[uint(from)].Nonce = nonce
								states[uint(from)].Balance -= balance
								states[uint(to)].Balance += balance
								block.Tansactions = append(block.Tansactions, string(ptx[:]))
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
			data := mevent.Data

			if len(n.Tetris.EventCh) < 100 { //这个地方有可能阻塞。。。
				n.Tetris.EventCh <- data
			} else {
				//fmt.Println("EventCh:", len(n.Tetris.EventCh))
			}

			//logging.Logger.Info("node receive ", mevent.MsgType, " ", mevent.From)
		case mtx := <-n.subscriberTx.MsgChan:
			//logging.Logger.Info("node receive ", mtx.MsgType, " ", mtx.From, " ", string(mtx.Data))
			//if len(n.Tetris.TxsCh) < 10000 { //这个地方有可能阻塞。。。
			var data common.Hash
			copy(data[:], mtx.Data)
			n.Tetris.TxsCh <- data
			//} else {
			//	//fmt.Println("Txs:", len(n.Tetris.TxsCh))
			//}

		case event := <-n.Tetris.SendEventCh:
			n.p2p.BroadcastMessage(p2p.Message{p2p.MessageTypeEvent, n.name, nil, event})
			h := sha256.Sum256(event)
			n.p2p.DhtSetValue(h[:], event)
			//logging.Logger.Info("send: ", event.Body.N)
		case hash := <-n.Tetris.RequestEventCh:
			data, err := n.p2p.DhtGetValue(hash[:])

			if err != nil {
				fmt.Println("dhtGetValue error")
			}

			go func(event []byte) {
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				n.Tetris.ParentEventCh <- data
			}(data)
		}
	}
}

//mock_core, 实现ICORE
func (n *Node) GetSigner() crypto.Signer {
	return NewMockSigner()
}

func (n *Node) GetPrivateKeyOfDefaultAccount() ([]byte, error) { //从node的accountManager取
	return hex.DecodeString(n.name)
}

func (n *Node) AddressFromPublicKey(publicKey []byte) ([]byte, error) {
	return publicKey, nil
}

type MockSigner struct {
	algrithm   crypto.Algorithm
	privateKey []byte
}

func NewMockSigner() *MockSigner {
	signer := &MockSigner{
		algrithm: crypto.ALG_UNKNOWN,
	}
	return signer
}

func (m *MockSigner) Algorithm() crypto.Algorithm {
	return m.algrithm
}

func (m *MockSigner) InitSigner(privateKey []byte) error {
	m.privateKey = privateKey
	return nil
}

func (m *MockSigner) Sign(data []byte) (signature *crypto.Signature, err error) {
	if m.privateKey == nil {
		logging.Logger.Warn("privateKey has not setted!")
		return nil, errors.New("privateKey has not setted")
	}

	signature = &crypto.Signature{
		Algorithm: m.Algorithm(),
		Signature: m.privateKey,
	}
	return signature, nil
}

func (m *MockSigner) RecoverPublicKey(data []byte, signature *crypto.Signature) (publicKey []byte, err error) {
	return signature.Signature, nil
}

func (m *MockSigner) Verify(publicKey []byte, data []byte, signature *crypto.Signature) bool {
	return true
}
