package main

import (
	"sync"
	//"log"

	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	cliMutex = &sync.Mutex{}
	setLogger()
	initFile := ".msshrc"
	run(initFile)
}

func main() {
	//app := cli.NewApp()
	//app.Name = "mssh"
	//app.Usage = "批量远程工具"
	//app.Version = "1.0.0"

	//app.Flags = []cli.Flag{
	//	cli.StringFlag{
	//		Name:        "conf",
	//		Value:       "conf/mssh.conf",
	//		Usage:       "指定配置文件",
	//		Destination: &conf,
	//	},
	//}

	//app.Run(os.Args)

	//s, err := connect("root", "123456", "127.0.0.1", 22)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//defer s.Close()
	//s.Stdout = os.Stdout
	log.Infoln("main start")
	if len(os.Args) > 1 {
		// 若带参数，则解释文件
		for _, file := range os.Args[1:] {
			run(file)
		}
	} else {
		interpret(os.Stdin)
	}
}
