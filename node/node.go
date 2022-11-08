/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package node

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/confile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
)

type Scheduler interface {
	Run()
}

type Node struct {
	Confile   confile.Confiler
	Chain     chain.Chainer
	Logs      logger.Logger
	Cache     db.Cacher
	Conn      *ConMgr
	lock      *sync.Mutex
	conns     *uint8
	FileDir   string
	TagDir    string
	FillerDir string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	var (
		err       error
		ok        bool
		remote    string
		tcpAddr   *net.TCPAddr
		listener  *net.TCPListener
		acceptTCP *net.TCPConn
	)

	// Start the subtask manager
	go n.CoroutineMgr()

	// Get an address of TCP end point
	tcpAddr, err = net.ResolveTCPAddr("tcp", ":"+n.Confile.GetServicePort())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Listen for TCP networks
	listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	time.Sleep(time.Second)
	log.Println("Service started successfully")

	for {
		// Connection interval
		time.Sleep(configs.TCP_Connection_Interval)

		// Accepts the next connection
		acceptTCP, err = listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("[err] The port is closed and the service exits.")
				os.Exit(1)
			}
			n.Logs.Common("error", fmt.Errorf("Accept tcp err: %v", err))
			continue
		}

		// Record client address
		remote = acceptTCP.RemoteAddr().String()
		n.Logs.Common("info", fmt.Errorf("Recv a conn: %v", remote))

		// Chain status
		if !n.Chain.GetChainStatus() {
			acceptTCP.Close()
			n.Logs.Common("info", fmt.Errorf("Chain state not available: %v", remote))
			continue
		}

		// Set server maximum connection control
		if n.GetConns() >= configs.MAX_TCP_CONNECTION {
			ok, err = n.Cache.Has([]byte(strings.Split(remote, ":")[0]))
			if ok {
				acceptTCP.Close()
				n.Logs.Common("info", fmt.Errorf("Conn full: %v", remote))
				continue
			}
		}

		// Start the processing service of the new connection
		go n.NewServer(NewTcp(acceptTCP)).Start()
	}
}

// InitLock is used to initialize lock
func (n *Node) InitLock() {
	n.lock = new(sync.Mutex)
	n.conns = new(uint8)
}

// AddConns is used to add a connection number record
func (n *Node) AddConns() {
	n.lock.Lock()
	*n.conns += 1
	n.lock.Unlock()
}

// ClearConns is used to clear a connection number record
func (n *Node) ClearConns() {
	n.lock.Lock()
	if *n.conns > 0 {
		*n.conns -= 1
	}
	n.lock.Unlock()
}

// ClearConns is used to clear a connection number record
func (n *Node) GetConns() uint8 {
	var num uint8
	n.lock.Lock()
	num = *n.conns
	n.lock.Unlock()
	return num
}
