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
	"cess-scheduler/configs"
	"cess-scheduler/internal/com"
	"cess-scheduler/internal/task"
	"cess-scheduler/pkg/chain"
	"cess-scheduler/pkg/configfile"
	"cess-scheduler/pkg/db"
	"cess-scheduler/pkg/logger"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
	"storj.io/common/base58"
)

// runCmd is used to start the scheduling service
func runCmd(cmd *cobra.Command, args []string) {
	var isReg bool
	// config file
	var configFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configFilePath = configpath1
	} else {
		configFilePath = configpath2
	}

	confile := configfile.NewConfigfile()
	if err := confile.Parse(configFilePath); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// chain client
	c, err := chain.NewChainClient(
		confile.GetRpcAddr(),
		confile.GetCtrlPrk(),
		time.Duration(time.Second*15),
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// judge the balance
	accountinfo, err := c.GetAccountInfo()
	if err != nil {
		log.Printf("Failed to get account information.\n")
		os.Exit(1)
	}
	if accountinfo.Data.Free.CmpAbs(
		new(big.Int).SetUint64(configs.MinimumBalance),
	) == -1 {
		log.Printf("Account balance is less than %v pico\n", configs.MinimumBalance)
		os.Exit(1)
	}

	// whether to register
	schelist, err := c.GetSchedulerInfo()
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
	} else {
		for _, v := range schelist {
			if v.ControllerUser == types.NewAccountID(c.GetPublicKey()) {
				isReg = true
				break
			}
		}
	}

	// register
	if !isReg {
		if err := register(confile, c); err != nil {
			os.Exit(1)
		}
	}

	// create data dir
	logDir, dbDir, fillerDir, err := creatDataDir(confile, c)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// cache
	db, err := db.NewLevelDB(dbDir, 0, 0, configs.NameSpace)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// logs
	var logs_info = make(map[string]string)
	for _, v := range configs.LogName {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	logs, err := logger.NewLogs(logs_info)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// run task
	go task.Run(confile, c, db, logs, fillerDir)
	com.Start(confile)
}

func register(confile configfile.Configfiler, c chain.Chainer) error {
	txhash, err := c.Register(
		confile.GetStashAcc(),
		base58.Encode(
			[]byte(confile.GetServiceAddr()+":"+confile.GetServicePort()),
		),
	)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Println("[err] Please check your wallet balance.")
		} else {
			if txhash != "" {
				msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
				msg += configs.HELP_register
				log.Printf("[pending] %v\n", msg)
			} else {
				log.Printf("[err] %v.\n", err)
			}
		}
		return err
	}
	log.Println("[ok] Registration success")
	return nil
}

func creatDataDir(
	confile configfile.Configfiler,
	c chain.Chainer,
) (string, string, string, error) {
	ctlAccount, err := c.GetCessAccount()
	if err != nil {
		return "", "", "", err
	}
	baseDir := filepath.Join(confile.GetDataDir(), ctlAccount, configs.BaseDir)
	log.Println(baseDir)
	_, err = os.Stat(baseDir)
	if err != nil {
		err = os.MkdirAll(baseDir, os.ModeDir)
		if err != nil {
			return "", "", "", err
		}
	}

	logDir := filepath.Join(baseDir, "log")
	if err := os.MkdirAll(logDir, os.ModeDir); err != nil {
		return "", "", "", err
	}

	dbDir := filepath.Join(baseDir, "db")
	if err := os.MkdirAll(dbDir, os.ModeDir); err != nil {
		return "", "", "", err
	}

	fillerDir := filepath.Join(baseDir, "filler")
	if err := os.MkdirAll(fillerDir, os.ModeDir); err != nil {
		return "", "", "", err
	}
	return logDir, dbDir, fillerDir, nil
}