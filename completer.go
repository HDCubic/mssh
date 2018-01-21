package main

import (
	"io/ioutil"

	"github.com/chzyer/readline"
)

// listFiles 列出某目录下的文件，以用作tab补全
func listFiles(path string) func(string) []string {
	return func(line string) []string {
		names := make([]string, 0)
		files, _ := ioutil.ReadDir(path)
		for _, f := range files {
			names = append(names, f.Name())
		}
		return names
	}
}

// completer 补全器
var completer = readline.NewPrefixCompleter(
	readline.PcItem("log",
		readline.PcItem("filename"),
	),
	readline.PcItem("put",
		readline.PcItemDynamic(listFiles("./"),
			readline.PcItem("remoteDir"),
		),
	),
	readline.PcItem("get",
		readline.PcItem("remotePath"),
	),
	readline.PcItem("connect",
		readline.PcItem("user"),
		readline.PcItem("password"),
		readline.PcItem("host"),
		readline.PcItem("port"),
	),
	readline.PcItem("release"),
	readline.PcItem("run",
		readline.PcItemDynamic(listFiles("./"),
			readline.PcItem("./"),
		),
	),
	readline.PcItem("check"),
	readline.PcItem("clear"),
	readline.PcItem("exit"),
)

// filterInput 过滤特定输入
func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}
