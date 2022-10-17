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
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
)

type Scheduler interface {
	Run()
}

type Node struct {
	Conn      *ConMgr
	Confile   configfile.Configfiler
	Chain     chain.Chainer
	Logs      logger.Logger
	Cache     db.Cacher
	FileDir   string
	TagDir    string
	FillerDir string
}

// New is used to build a node instance
func New() *Node {
	return &Node{}
}

func (n *Node) Run() {
	go n.CoroutineMgr()
	// Get an address of TCP end point
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+n.Confile.GetServicePort())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Listen for TCP networks
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	for {
		// Accepts the next connection
		acceptTCP, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("[err] The port is closed and the service exits.")
				os.Exit(1)
			}
			n.Logs.Log("common", "error", fmt.Errorf("accept tcp: %v\n", err))
			continue
		}

		remote := acceptTCP.RemoteAddr().String()
		n.Logs.Log("common", "info", fmt.Errorf("received a conn: %v\n", remote))

		// Set server maximum connection control
		if TCP_ConnLength.Load() > configs.MAX_TCP_CONNECTION {
			acceptTCP.Close()
			n.Logs.Log("common", "info", fmt.Errorf("close conn: %v\n", remote))
			continue
		}

		// Start the processing service of the new connection
		go n.NewServer(NewTcp(acceptTCP), n.FileDir).Start()
		time.Sleep(time.Millisecond)
	}
}