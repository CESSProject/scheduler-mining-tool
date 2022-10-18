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
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/CESSProject/cess-scheduler/api/protobuf"
	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// task_ValidateProof is used to verify the proof data
func (node *Node) task_ValidateProof(ch chan bool) {
	var (
		err         error
		goeson      bool
		txhash      string
		poDR2verify pbc.PoDR2Verify
		reqtag      protobuf.ReadTagReq
		proofs      = make([]chain.Proof, 0)
	)
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	node.Logs.Verify("info", errors.New(">>> Start task_ValidateProof <<<"))

	reqtag.Acc = node.Chain.GetPublicKey()

	for {
		proofs, err = node.Chain.GetProofs()
		if err != nil {
			if err.Error() != chain.ERR_RPC_EMPTY_VALUE.Error() {
				node.Logs.Verify("error", err)
			}
		}
		if len(proofs) == 0 {
			time.Sleep(time.Minute)
			continue
		}

		node.Logs.Verify("info", fmt.Errorf("There are %d proofs", len(proofs)))

		var respData []byte
		var tag pbc.TagInfo
		var verifyResults = make([]chain.ProofResult, 0)
		for i := 0; i < len(proofs); i++ {
			if len(verifyResults) >= configs.Max_SubProofResults {
				txhash = ""
				for txhash == "" {
					txhash, _ = node.Chain.SubmitProofResults(verifyResults)
					if txhash != "" {
						node.Logs.Verify("info", fmt.Errorf("Proof result submitted: %v", txhash))
						break
					}
					time.Sleep(configs.BlockInterval)
				}
				verifyResults = make([]chain.ProofResult, 0)
			}
			goeson = false
			addr, err := utils.EncodePublicKeyAsCessAccount(proofs[i].Miner_pubkey[:])
			if err != nil {
				node.Logs.Log("verified", "error", errors.Errorf("%v,%v", proofs[i].Miner_pubkey, err))
			}

			cacheData, err := node.Cache.Get(proofs[i].Miner_pubkey[:])
			if err != nil {
				resultTemp := chain.ProofResult{}
				resultTemp.PublicKey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				if err.Error() == "leveldb: not found" {
					resultTemp.Result = false
				} else {
					resultTemp.Result = true
				}
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			var minerinfo chain.Cache_MinerInfo
			err = json.Unmarshal(cacheData, &minerinfo)
			if err != nil {
				node.Logs.Log("verified", "error", errors.Errorf("%v,%v", addr, err))
				resultTemp := chain.ProofResult{}
				resultTemp.PublicKey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Result = true
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			goeson = false
			// reqtag.FileId = string(proofs[i].Challenge_info.File_id)
			// req_proto, err := proto.Marshal(&reqtag)
			// if err != nil {
			// 	logs.Log("vp", "error", errors.Errorf("%v,%v", addr, err))
			// }
			// for j := 0; j < 3; j++ {
			// 	respData, err = rpc.WriteData(
			// 		string(minerinfo.Ip),
			// 		com.RpcService_Miner,
			// 		com.RpcMethod_Miner_ReadFileTag,
			// 		time.Duration(time.Second*30),
			// 		req_proto,
			// 	)
			// 	if err != nil {
			// 		logs.Log("vp", "error", errors.Errorf("%v,%v", addr, err))
			// 		time.Sleep(time.Second * time.Duration(utils.RandomInRange(3, 6)))
			// 	} else {
			// 		goeson = true
			// 		break
			// 	}
			// }

			if !goeson {
				resultTemp := chain.ProofResult{}
				resultTemp.PublicKey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Result = true
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			err = json.Unmarshal(respData, &tag)
			if err != nil {
				node.Logs.Log("verified", "error", errors.Errorf("%v,%v", addr, err))
			}
			qSlice, err := pbc.PoDR2ChallengeGenerateFromChain(
				proofs[i].Challenge_info.Block_list,
				proofs[i].Challenge_info.Random,
			)
			if err != nil {
				node.Logs.Log("verified", "error",
					errors.Errorf("[%v] [%v] [%v] qslice: %v",
						addr,
						len(proofs[i].Challenge_info.Block_list),
						len(proofs[i].Challenge_info.Random),
						err,
					))
			}

			poDR2verify.QSlice = qSlice
			poDR2verify.MU = make([][]byte, len(proofs[i].Mu))
			for j := 0; j < len(proofs[i].Mu); j++ {
				poDR2verify.MU[j] = append(poDR2verify.MU[j], proofs[i].Mu[j]...)
			}

			poDR2verify.Sigma = proofs[i].Sigma
			poDR2verify.T = tag.T

			result := poDR2verify.PoDR2ProofVerify(pbc.Key_SharedG, pbc.Key_Spk, string(pbc.Key_SharedParams))
			resultTemp := chain.ProofResult{}
			resultTemp.PublicKey = proofs[i].Miner_pubkey
			resultTemp.FileId = proofs[i].Challenge_info.File_id
			resultTemp.Result = types.Bool(result)
			verifyResults = append(verifyResults, resultTemp)
		}
		if len(verifyResults) > 0 {
			txhash = ""
			for txhash == "" {
				txhash, _ = node.Chain.SubmitProofResults(verifyResults)
				if txhash != "" {
					node.Logs.Log("verified", "info", errors.Errorf("Proof result submitted: %v", txhash))
					break
				}
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(3, 15)))
			}
			verifyResults = make([]chain.ProofResult, 0)
		}
		time.Sleep(time.Second * configs.BlockInterval)
		runtime.GC()
	}
}
