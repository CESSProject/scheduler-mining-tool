/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

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
	"encoding/json"
	"fmt"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

// task_MinerCache obtains the miners' information on the chain
// and records it to the cache.
func (n *Node) task_minerCache(ch chan<- bool) {
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
		ch <- true
	}()

	var (
		ipv4       string
		minerCache chain.Cache_MinerInfo
		minerInfo  chain.MinerInfo
	)

	n.Logs.MinerCache("info", errors.New(">>> Start task_minerCache <<<"))

	for {
		for n.Chn.GetChainStatus() {
			// Get the account public key of all miners
			allMinerAcc, _ := n.Chn.GetAllStorageMiner()
			if len(allMinerAcc) == 0 {
				time.Sleep(configs.BlockInterval)
				continue
			}

			for i := 0; i < len(allMinerAcc); i++ {
				// CESS addr
				addr, err := utils.EncodePublicKeyAsCessAccount(allMinerAcc[i][:])
				if err != nil {
					n.Logs.MinerCache("error", fmt.Errorf("%v, %v", allMinerAcc[i], err))
					continue
				}

				// Get the details of miners
				minerInfo, err = n.Chn.GetStorageMinerInfo(allMinerAcc[i][:])
				if err != nil {
					n.Logs.MinerCache("error", fmt.Errorf("[%v] %v", addr, err))
					continue
				}

				// if exit
				if string(minerInfo.State) != chain.MINER_STATE_POSITIVE {
					exist, _ := n.Cach.Has(allMinerAcc[i][:])
					if exist {
						n.Cach.Delete(allMinerAcc[i][:])
					}
					continue
				}

				// save data
				ipv4 = fmt.Sprintf("%d.%d.%d.%d",
					minerInfo.Ip.Value[0],
					minerInfo.Ip.Value[1],
					minerInfo.Ip.Value[2],
					minerInfo.Ip.Value[3],
				)
				minerCache.Ip = fmt.Sprintf("%v:%d", ipv4, minerInfo.Ip.Port)
				minerCache.Free = minerInfo.Idle_space.Uint64()
				value, err := json.Marshal(&minerCache)
				if err != nil {
					n.Logs.MinerCache("error", fmt.Errorf("[%v] %v", addr, err))
					continue
				}

				// save or update cache
				err = n.Cach.Put(allMinerAcc[i][:], value)
				if err != nil {
					n.Logs.MinerCache("error", fmt.Errorf("[%v] %v", addr, err))
					continue
				}

				err = n.Cach.Put([]byte(ipv4), nil)
				if err != nil {
					n.Logs.MinerCache("error", fmt.Errorf("[%v] %v", addr, err))
					continue
				}
				n.Logs.MinerCache("info", fmt.Errorf("[%v] %v : %v : %v", addr, ipv4, minerInfo.Ip.Port, minerCache.Free))
			}
			time.Sleep(time.Minute * 10)
		}
		time.Sleep(configs.BlockInterval)
	}
}
