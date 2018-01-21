package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Client 一个ssh客户端的相关参数集合
type Client struct {
	Cli      *ssh.Client
	HomePath string
}

var (
	cliMutex *sync.Mutex
	cliMap   = map[string]*Client{}
	wg       sync.WaitGroup
)

// 启动一个并行任务
func launch(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}

// done 内置命令，等待并行任务完成
func done() {
	wg.Wait()
	log.Info("multi command done")
}

// connect 内置命令，连接服务器
func connect(user, password, host, port string) {
	launch(func() {
		// 此处为了防止建立连接耗时过长的情况，另开一个线程去处理
		if _, ok := cliMap[host]; ok {
			// 已连接的服务器不再重复连接
			log.Infof("[%s] connected", host)
			return
		}
		if port == "" {
			port = "22"
		}
		var (
			auth         []ssh.AuthMethod
			addr         string
			clientConfig *ssh.ClientConfig
			client       *ssh.Client
			//session      *ssh.Session
			err error
		)
		// get auth method
		auth = make([]ssh.AuthMethod, 0)
		auth = append(auth, ssh.Password(password))

		clientConfig = &ssh.ClientConfig{
			User:    user,
			Auth:    auth,
			Timeout: 30 * time.Second,
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}

		addr = fmt.Sprintf("%s:%s", host, port)

		if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
			log.Errorf("[%s] ssh Dial error: %s", host, err)
			return
		}
		session, err := client.NewSession()
		if err != nil {
			log.Errorf("[%s] get session error: %s", host, err.Error())
			return
		}
		homePath, err := session.Output("pwd")
		if err != nil {
			log.Errorf("[%s] get home path error: %s", host, err.Error())
			return
		}
		//session.Stdout = os.Stdout
		cli := &Client{
			Cli:      client,
			HomePath: strings.TrimSpace(string(homePath)),
		}

		cliMutex.Lock()
		defer cliMutex.Unlock()
		cliMap[host] = cli
		log.Infof("[%s] connect success", host)
	})
}

// release 内置命令，释放连接
func release(host string) {
	if client, ok := cliMap[host]; ok {
		err := client.Cli.Close()
		if err != nil {
			log.Warnf("[%s] close error: %s", host, err)
		}
		delete(cliMap, host)
		log.Warnf("[%s] released", host)
		return
	}
	log.Warnf("[%s] has not connected yet", host)
}

// put 内置命令，批量上传文件
func put(file string, dstDir string) {
	//fmt.Println("func put:", file, toDir)
	for host, client := range cliMap {
		toDir := dstDir
		if toDir == "" {
			toDir = client.HomePath
		}
		sftpClient, err := sftp.NewClient(client.Cli)
		if err != nil {
			log.Errorf("[%s] get sftpClient error: %s", host, err)
			continue
		}
		defer sftpClient.Close()

		srcFile, err := os.Open(file)
		if err != nil {
			log.Errorf("[%s] open srcFile error: %s", host, err)
			continue
		}
		defer srcFile.Close()

		remoteFileName := path.Base(file)
		remotePath := path.Join(toDir, remoteFileName)
		dstFile, err := sftpClient.Create(remotePath)
		if err != nil {
			log.Errorf("[%s] create dstFile error: %s", host, err)
			continue
		}
		defer dstFile.Close()

		buf := make([]byte, 1024)
		for {
			n, _ := srcFile.Read(buf)
			if n == 0 {
				break
			}
			dstFile.Write(buf[0:n])
		}
		log.Infof("[%s] put file [%s] to [%s] success", host, file, remotePath)

		//session, err := client.NewSession()
		//session.Stdout = os.Stdout
		//if err != nil {
		//	log.Println(host, "get session error:", err)
		//	continue
		//}
		//fmt.Println(host, session)
		//session.Close()
	}
}

// get 内置命令，批量下载文件
func get(file string) {
	for host, client := range cliMap {
		sftpClient, err := sftp.NewClient(client.Cli)
		if err != nil {
			log.Errorf("[%s] get sftpClient error: %s", host, err)
			continue
		}
		defer sftpClient.Close()

		srcFile, err := sftpClient.Open(file)
		if err != nil {
			log.Errorf("[%s] open srcFile error: %s", host, err)
			continue
		}
		defer srcFile.Close()

		localFileName := path.Base(file)
		localDir := path.Join(".", "download", host)
		os.MkdirAll(localDir, 0777)
		localPath := path.Join(localDir, localFileName)
		dstFile, err := os.Create(localPath)
		if err != nil {
			log.Errorf("[%s] create dstFile error: %s", host, err)
			continue
		}
		defer dstFile.Close()

		buf := make([]byte, 1024)
		for {
			n, _ := srcFile.Read(buf)
			if n == 0 {
				break
			}
			dstFile.Write(buf[0:n])
		}
		log.Infof("[%s] get file [%s] to [%s] success", host, file, localPath)

		//session, err := client.NewSession()
		//session.Stdout = os.Stdout
		//if err != nil {
		//	log.Println(host, "get session error:", err)
		//	continue
		//}
		//fmt.Println(host, session)
		//session.Close()
	}
}

// check 内置命令，检测已建立连接
func check() {
	for host, client := range cliMap {
		_ = client
		log.Infof("[%s] connecting", host)
	}
}

// remote 内置命令，批量远程执行
func remote(cmd string) {
	for host, client := range cliMap {
		session, err := client.Cli.NewSession()
		session.Stdout = os.Stdout
		if err != nil {
			log.Errorf("[%s] get session error: %s", host, err.Error())
			continue
		}
		err = session.Run(cmd)
		if err != nil {
			// 此次执行有错误
			log.Errorf("[%s] remote command [%s] failed: %s", host, cmd, err.Error())
			continue
		}
		log.Infof("[%s] remote command [%s] success", host, cmd)
		session.Close()
	}
}

// run 内置命令，执行mssh脚本
func run(script string) {
	fp, err := os.Open(script)
	if err != nil {
		log.Errorf("run script %s error: %s", script, err.Error())
		return
	}
	defer fp.Close()
	interpret(fp)

}

// addLogger 内置命令，增加日志记录
func addLogger(file string) {
	lfHook := lfshook.NewHook(
		file,
		&LogFormatter{
			EnableTime:      true,
			EnablePos:       true,
			EnableColor:     true,
			TimestampFormat: "2006-01-02 15:04:05",
			CallerLevel:     10,
		})
	log.AddHook(lfHook)
}

// setLogger 设置默认日志格式
func setLogger() {
	log.SetFormatter(&LogFormatter{
		EnableColor:     true,
		TimestampFormat: "",
		CallerLevel:     7,
	})
	log.SetOutput(os.Stdout)
}

// clear 内置命令，清屏
func clear() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// exit 退出mssh命令行
func exit() {
	for host, client := range cliMap {
		err := client.Cli.Close()
		if err != nil {
			log.Errorf("[%s] close error: %s", host, err.Error())
			continue
		}
		log.Infof("[%s] closed", host)
	}
	os.Exit(0)
}
