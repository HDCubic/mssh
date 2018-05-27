package main

import (
	"flag"
	"fmt"
	_ "mssh/pkg/cmd"
	"mssh/pkg/interpreter"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	// VERSION 版本信息
	VERSION string
	// BUILD 构建时间
	BUILD string
	// COMMITSHA1 git commit ID
	COMMITSHA1 string
)

//命令行选项
var (
	h = flag.Bool("h", false, "Show this help")
	v = flag.Bool("v", false, "Show version")
)

func init() {
	setLogger()
}

func usage() {
	fmt.Println(`Usage:
	mssh [-hv]
	mssh [filename] [filename] ...
	mssh
	`)
	fmt.Printf(`Info:
	version: %s
	release time: %s
	commit sha1: %s
	`, VERSION, BUILD, COMMITSHA1)
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	if *h {
		usage()
		return
	}
	if *v {
		fmt.Println(VERSION)
		return
	}
	// 加载初始化配置
	initFile := ".msshrc"
	interpreter.Run(initFile)

	log.Infoln("main start")
	if len(os.Args) > 1 {
		// 若带参数，则解释文件
		for _, file := range os.Args[1:] {
			interpreter.Run(file)
		}
	} else {
		interpreter.Interpret(os.Stdin)
	}
}
