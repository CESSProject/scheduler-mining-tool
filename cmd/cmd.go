package cmd

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/db"
	"cess-scheduler/internal/logger"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/rpc"
	"cess-scheduler/internal/task"
	"cess-scheduler/tools"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"storj.io/common/base58"
)

const (
	Name        = "cess-scheduler"
	Description = "Implementation of Scheduling Service for Consensus Nodes"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   Name,
	Short: Description,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// init
func init() {
	rootCmd.AddCommand(
		Command_Default(),
		Command_Version(),
		Command_Register(),
		Command_Run(),
		Command_Update(),
	)
	rootCmd.PersistentFlags().StringVarP(&configs.ConfigFilePath, "config", "c", "", "Custom profile")
}

func Command_Version() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "version",
		Short:                 "Print version information",
		Run:                   Command_Version_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Default() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "default",
		Short:                 "Generate profile template",
		Run:                   Command_Default_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Register() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "register",
		Short:                 "Register scheduler information to the chain",
		Run:                   Command_Register_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Update() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "update <ip> <port>",
		Short:                 "Update scheduling service ip and port",
		Run:                   Command_Update_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Run() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Operation scheduling service",
		Run:                   Command_Run_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

// Print version number and exit
func Command_Version_Runfunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Version)
	os.Exit(0)
}

// Generate configuration file template
func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(configs.ConfigFile_Templete, configs.DefaultConfigurationFileName)
	pwd, err := os.Getwd()
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	path := filepath.Join(pwd, configs.DefaultConfigurationFileName)
	log.Printf("\x1b[%dm[ok]\x1b[0m %v\n", 42, path)
	os.Exit(0)
}

// Scheduler registration
func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	chain.ChainInit()
	register()
}

// start service
func Command_Run_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	chain.ChainInit()
	flag := register_if()
	if !flag {
		logger.Logger_Init()
	}
	db.Init()
	task.Run()
	rpc.Rpc_Main()
}

// Scheduler registration function
func register() {
	sd, err := chain.GetSchedulerInfoOnChain()
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			log.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
			os.Exit(1)
		}
	}
	for _, v := range sd {
		if v.ControllerUser == types.NewAccountID(configs.PublicKey) {
			log.Printf("\x1b[%dm[ok]\x1b[0m The account is already registered.\n", 42)
			os.Exit(0)
		}
	}
	rgst()
	os.Exit(0)
}

func register_if() bool {
	var reg bool
	sd, err := chain.GetSchedulerInfoOnChain()
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			rgst()
			return true
		}
		log.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}

	for _, v := range sd {
		if v.ControllerUser == types.NewAccountID(configs.PublicKey) {
			reg = true
		}
	}
	if !reg {
		rgst()
		return true
	}

	log.Printf("\x1b[%dm[ok]\x1b[0m Registered schedule\n", 42)

	addr, err := chain.GetAddressByPrk(configs.C.CtrlPrk)
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	baseDir := filepath.Join(configs.C.DataDir, addr, configs.BaseDir)

	configs.LogFileDir = filepath.Join(baseDir, configs.LogFileDir)
	log.Printf(configs.LogFileDir)
	err = os.RemoveAll(configs.LogFileDir)
	if err != nil {
		log.Println(err)
	}
	if err = tools.CreatDirIfNotExist(configs.LogFileDir); err != nil {
		goto Err
	}
	//
	configs.FileCacheDir = filepath.Join(baseDir, configs.FileCacheDir)
	log.Printf(configs.FileCacheDir)
	err = os.RemoveAll(configs.FileCacheDir)
	if err != nil {
		log.Println(err)
	}
	if err = tools.CreatDirIfNotExist(configs.FileCacheDir); err != nil {
		goto Err
	}
	//
	configs.DbFileDir = filepath.Join(baseDir, configs.DbFileDir)
	log.Printf(configs.DbFileDir)
	err = os.RemoveAll(configs.DbFileDir)
	if err != nil {
		log.Println(err)
	}
	if err = tools.CreatDirIfNotExist(configs.DbFileDir); err != nil {
		goto Err
	}
	//
	configs.SpaceCacheDir = filepath.Join(baseDir, configs.SpaceCacheDir)
	log.Printf(configs.SpaceCacheDir)
	err = os.RemoveAll(configs.SpaceCacheDir)
	if err != nil {
		log.Println(err)
	}
	if err = tools.CreatDirIfNotExist(configs.SpaceCacheDir); err != nil {
		goto Err
	}

	return false
Err:
	log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	os.Exit(1)
	return false
}

func rgst() {
	addr, err := chain.GetAddressByPrk(configs.C.CtrlPrk)
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	res := base58.Encode([]byte(configs.C.ServiceAddr + ":" + configs.C.ServicePort))

	txhash, err := chain.RegisterToChain(
		configs.C.CtrlPrk,
		configs.C.StashAcc,
		res,
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
		os.Exit(1)
	}
	log.Printf("\x1b[%dm[ok]\x1b[0m Registration success\n", 42)

	baseDir := filepath.Join(configs.C.DataDir, addr, configs.BaseDir)
	log.Println(baseDir)
	err = os.RemoveAll(baseDir)
	if err != nil {
		log.Panicln(err)
	}
	configs.LogFileDir = filepath.Join(baseDir, configs.LogFileDir)
	if err = tools.CreatDirIfNotExist(configs.LogFileDir); err != nil {
		goto Err
	}
	configs.FileCacheDir = filepath.Join(baseDir, configs.FileCacheDir)
	if err = tools.CreatDirIfNotExist(configs.FileCacheDir); err != nil {
		goto Err
	}
	configs.DbFileDir = filepath.Join(baseDir, configs.DbFileDir)
	if err = tools.CreatDirIfNotExist(configs.DbFileDir); err != nil {
		goto Err
	}
	configs.SpaceCacheDir = filepath.Join(baseDir, configs.SpaceCacheDir)
	if err = tools.CreatDirIfNotExist(configs.SpaceCacheDir); err != nil {
		goto Err
	}
	logger.Logger_Init()
	Com.Sugar().Infof("Registration message:")
	Com.Sugar().Infof("ChainAddr:%v", configs.C.RpcAddr)
	Com.Sugar().Infof("ServiceAddr:%v", res)
	Com.Sugar().Infof("DataDir:%v", configs.C.DataDir)
	Com.Sugar().Infof("ControllerAccountPhrase:%v", configs.C.CtrlPrk)
	Com.Sugar().Infof("StashAccountAddress:%v", configs.C.StashAcc)
	return
Err:
	log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	os.Exit(1)
}

// Schedule update ip function
func Command_Update_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	if len(os.Args) >= 4 {
		_, err := strconv.Atoi(os.Args[3])
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m Please fill in the correct port number.\n", 41)
			os.Exit(1)
		}
		res := base58.Encode([]byte(os.Args[2] + ":" + os.Args[3]))
		txhash, err := chain.UpdatePublicIp(configs.C.CtrlPrk, res)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				log.Println("[err] Please check your wallet balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
					msg += configs.HELP_update
					log.Printf("[pending] %v\n", msg)
				} else {
					log.Printf("[err] %v.\n", err)
				}
			}
			os.Exit(1)
		}
		log.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
		os.Exit(0)
	}
	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'scheduler update ip port'\n", 41)
	os.Exit(1)
}

// Parse the configuration file
func refreshProfile(cmd *cobra.Command) {
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configs.ConfigFilePath = configpath1
	} else {
		configs.ConfigFilePath = configpath2
	}
	parseProfile()
}

func parseProfile() {
	var (
		err          error
		confFilePath string
	)
	if configs.ConfigFilePath == "" {
		confFilePath = "./conf.toml"
	} else {
		confFilePath = configs.ConfigFilePath
	}

	f, err := os.Stat(confFilePath)
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m The '%v' file does not exist\n", 41, confFilePath)
		os.Exit(1)
	}
	if f.IsDir() {
		log.Printf("\x1b[%dm[err]\x1b[0m The '%v' is not a file\n", 41, confFilePath)
		os.Exit(1)
	}

	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
		os.Exit(1)
	}
	err = viper.Unmarshal(configs.C)
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(1)
	}

	if configs.C.CtrlPrk == "" ||
		configs.C.DataDir == "" ||
		configs.C.RpcAddr == "" ||
		configs.C.ServiceAddr == "" ||
		configs.C.StashAcc == "" {
		log.Printf("\x1b[%dm[err]\x1b[0m The configuration file cannot have empty entries.\n", 41)
		os.Exit(1)
	}

	if configs.C.ServicePort != "" {
		port, err := strconv.Atoi(configs.C.ServicePort)
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m Please fill in the correct port number.\n", 41)
			os.Exit(1)
		}
		if port < 1024 {
			log.Printf("\x1b[%dm[err]\x1b[0m Prohibit the use of system reserved port: %v.\n", 41, port)
			os.Exit(1)
		}
		if port > 65535 {
			log.Printf("\x1b[%dm[err]\x1b[0m The port number cannot exceed 65535.\n", 41)
			os.Exit(1)
		}
	} else {
		log.Printf("\x1b[%dm[err]\x1b[0m Please set the port number.\n", 41)
		os.Exit(1)
	}

	err = tools.CreatDirIfNotExist(configs.C.DataDir)
	if err != nil {
		log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	//
	configs.PublicKey, err = chain.GetPublicKeyByPrk(configs.C.CtrlPrk)
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
}
