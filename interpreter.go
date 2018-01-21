package main

import (
	"io"
	"reflect"
	"strings"

	"github.com/chzyer/readline"
	"github.com/cosiner/argv"
	log "github.com/sirupsen/logrus"
)

// funcMap 命令方法映射
var funcMap = map[string]interface{}{
	"put":     put,
	"get":     get,
	"check":   check,
	"clear":   clear,
	"done":    done,
	"connect": connect,
	"release": release,
	"log":     addLogger,
	//"run":     run,
	"exit": exit,
}

// call 内置命令通用调用方法
func call(m map[string]interface{}, name string, params ...interface{}) {
	ft := reflect.TypeOf(m[name])
	extra := ft.NumIn() - len(params)
	if extra > 0 {
		// 函数需求参数多余提供参数的情况
		for i := 0; i < extra; i++ {
			params = append(params, "")
		}
	} else if extra < 0 {
		// 函数需求参数少于提供的参数的情况
		log.Warn("params count not match")
		return
	}
	f := reflect.ValueOf(m[name])
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		//fmt.Println(param)
		in[k] = reflect.ValueOf(param)
	}
	f.Call(in)
}

// interpret 解释输入流
func interpret(in io.ReadCloser) {
	readline.Stdin = in
	r, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[33m[mssh ~ ]#\033[0m ",
		HistoryFile:     "/tmp/mssh.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer r.Close()

	//log.SetOutput(r.Stderr())
	for {
		line, err := r.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		//parseLine(line)
		if len(line) == 0 {
			// 输入为空继续
			continue
		}
		if strings.HasPrefix(line, "#") {
			// 将"#"定为注释
			continue
		}
		args, err := argv.Argv([]rune(line), map[string]string{}, argv.Run)
		if err != nil {
			// 解析错误打日志继续
			log.Error(line, err)
			continue
		}
		if _, ok := funcMap[args[0][0]]; ok {
			params := []interface{}{}
			if len(args[0]) > 1 {
				for _, param := range args[0][1:] {
					params = append(params, param)
				}
			}
			call(funcMap, args[0][0], params...)
		} else if strings.HasPrefix(line, "run") {
			// 对run特殊处理，放在funcMap中会出现循环调用
			if len(args[0]) == 1 {
				continue
			}
			for _, param := range args[0][1:] {
				run(param)
			}
		} else {
			remote(line)
		}
	}
}
