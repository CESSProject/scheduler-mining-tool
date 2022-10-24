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

package console

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/node"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/btcsuite/btcutil/base58"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
//
// Usage:
//
//	scheduler run
func runCmd(cmd *cobra.Command, args []string) {
	var (
		err      error
		logDir   string
		cacheDir string
		node     = node.New()
	)

	node.Confile, err = buildConfigFile(cmd)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	node.Chain, err = buildChain(node.Confile, configs.TimeOut_WaitBlock)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	node.ChainStatus = &atomic.Bool{}
	node.ChainStatus.Store(true)
	node.Connections = &atomic.Uint32{}
	node.Connections.Store(0)

	// create data dir
	logDir, cacheDir, node.FillerDir, node.FileDir, node.TagDir, err = buildDir(node.Confile, node.Chain)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// cache
	node.Cache, err = buildCache(cacheDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// logs
	node.Logs, err = buildLogs(logDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// run
	node.Run()
}

func buildConfigFile(cmd *cobra.Command) (configfile.Configfiler, error) {
	var configFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configFilePath = configpath1
	} else {
		configFilePath = configpath2
	}

	cfg := configfile.NewConfigfile()
	if err := cfg.Parse(configFilePath); err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildChain(cfg configfile.Configfiler, timeout time.Duration) (chain.Chainer, error) {
	var isReg bool
	// connecting chain
	client, err := chain.NewChainClient(cfg.GetRpcAddr(), cfg.GetCtrlPrk(), timeout)
	if err != nil {
		return nil, err
	}

	// judge the balance
	accountinfo, err := client.GetAccountInfo(client.GetPublicKey())
	if err != nil {
		return nil, err
	}

	if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(configs.MinimumBalance)) == -1 {
		return nil, fmt.Errorf("Account balance is less than %v pico\n", configs.MinimumBalance)
	}

	// sync block
	for {
		ok, err := client.GetSyncStatus()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		log.Println("In sync block...")
		time.Sleep(time.Second * configs.BlockInterval)
	}
	log.Println("Sync complete")

	// whether to register
	schelist, err := client.GetAllSchedulerInfo()
	if err != nil && err.Error() != chain.ERR_RPC_EMPTY_VALUE.Error() {
		return nil, err
	}

	for _, v := range schelist {
		if v.ControllerUser == client.NewAccountId(client.GetPublicKey()) {
			isReg = true
			break
		}
	}

	// register
	if !isReg {
		if err := register(cfg, client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func register(cfg configfile.Configfiler, client chain.Chainer) error {
	txhash, err := client.Register(
		cfg.GetStashAcc(), base58.Encode([]byte(cfg.GetServiceAddr()+":"+cfg.GetServicePort())),
	)
	if err != nil {
		if err.Error() == chain.ERR_RPC_EMPTY_VALUE.Error() {
			return fmt.Errorf("[err] Please check your wallet balance")
		} else {
			if txhash != "" {
				msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
				msg += configs.HELP_register
				return fmt.Errorf("[pending] %v\n", msg)
			}
			return err
		}
	}
	return nil
}

func buildDir(cfg configfile.Configfiler, client chain.Chainer) (string, string, string, string, string, error) {
	ctlAccount, err := client.GetCessAccount()
	if err != nil {
		return "", "", "", "", "", err
	}
	baseDir := filepath.Join(cfg.GetDataDir(), ctlAccount, configs.BaseDir)
	log.Println(baseDir)
	_, err = os.Stat(baseDir)
	if err != nil {
		err = os.MkdirAll(baseDir, os.ModeDir)
		if err != nil {
			return "", "", "", "", "", err
		}
	}

	logDir := filepath.Join(baseDir, configs.LogDir)
	_, err = os.Stat(logDir)
	if err == nil {
		bkp := logDir + fmt.Sprintf("_%v", time.Now().Unix())
		os.Rename(logDir, bkp)
	}
	if err := os.MkdirAll(logDir, os.ModeDir); err != nil {
		return "", "", "", "", "", err
	}

	cacheDir := filepath.Join(baseDir, configs.CacheDir)
	os.RemoveAll(cacheDir)
	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return "", "", "", "", "", err
	}

	fillerDir := filepath.Join(baseDir, configs.FillerDir)
	os.RemoveAll(fillerDir)
	if err := os.MkdirAll(fillerDir, os.ModeDir); err != nil {
		return "", "", "", "", "", err
	}

	fileDir := filepath.Join(baseDir, configs.FileDir)
	os.RemoveAll(fileDir)
	if err := os.MkdirAll(fileDir, os.ModeDir); err != nil {
		return "", "", "", "", "", err
	}

	tagDir := filepath.Join(baseDir, configs.TagDir)
	os.RemoveAll(tagDir)
	if err := os.MkdirAll(tagDir, os.ModeDir); err != nil {
		return "", "", "", "", "", err
	}
	return logDir, cacheDir, fillerDir, fileDir, tagDir, nil
}

func buildCache(cacheDir string) (db.Cacher, error) {
	return db.NewCache(cacheDir, 0, 0, configs.NameSpace)
}

func buildLogs(logDir string) (logger.Logger, error) {
	var logs_info = make(map[string]string)
	for _, v := range configs.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}
