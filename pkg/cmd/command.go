package cmd

import (
	"fmt"
	"mssh/pkg/reg"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
)

func init() {
	cliMutex = &sync.Mutex{}
	reg.Regist("inner", "vi", Vim, "启用vim编辑器", `vim <filename>`, []*reg.Param{
		&reg.Param{Name: "filename", Type: "string", Necessity: true, Desc: "文件名"},
	})
	reg.Regist("inner", "vim", Vim, "启用vim编辑器", `vim <filename>`, []*reg.Param{
		&reg.Param{Name: "filename", Type: "string", Necessity: true, Desc: "文件名"},
	})
	reg.Regist("inner", "clear", Clear, "清屏", `clear`, []*reg.Param{})
	reg.Regist("inner", "exit", Exit, "退出命令行界面", `exit`, []*reg.Param{})
	reg.Regist("inner", "done", Done, "等待批量任务完成", `done`, []*reg.Param{})

	reg.Regist("file", "put", Put, "批量上传文件", `put <filePath> <remoteDir>`, []*reg.Param{
		&reg.Param{Name: "filePath", Type: "string", Necessity: true, Desc: "本地文件路径"},
		&reg.Param{Name: "remoteDir", Type: "string", Necessity: false, Desc: "远程目录, 默认用户home目录, 如: /root/"},
	})
	reg.Regist("file", "get", Get, "批量下载文件, 本操作会将文件下载到执行目录下的download目录下服务器地址对应目录中", `get <remotePath>`, []*reg.Param{
		&reg.Param{Name: "remotePath", Type: "string", Necessity: true, Desc: "远程文件路径"},
	})

	reg.Regist("conn", "check", Check, "检查连接状态", `check`, []*reg.Param{})
	reg.Regist("conn", "connect", Connect, "连接远程主机", `connect <username> <password> <host> <port> <timeout>`, []*reg.Param{
		&reg.Param{Name: "username", Type: "string", Necessity: true, Desc: "用户名"},
		&reg.Param{Name: "password", Type: "string", Necessity: true, Desc: "密码"},
		&reg.Param{Name: "host", Type: "string", Necessity: true, Desc: "服务器地址"},
		&reg.Param{Name: "port", Type: "int", Necessity: false, Desc: "sshd服务端口, 默认 22"},
		&reg.Param{Name: "timeout", Type: "int", Necessity: false, Desc: "连接超时时间(单位 s), 默认 5 "},
	})
	reg.Regist("conn", "release", Release, "释放远程连接", `release <host>`, []*reg.Param{
		&reg.Param{Name: "host", Type: "string", Necessity: true, Desc: "服务器地址"},
	})
}

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

// Done 内置命令，等待并行任务完成
func Done() {
	wg.Wait()
	log.Info("multi command done")
}

// Connect 内置命令，连接服务器
func Connect(user, password, host, port, timeout string) {
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
		if timeout == "" {
			timeout = "5"
		}
		t, err := strconv.Atoi(timeout)
		if err != nil {
			log.Errorf("[%s] convert timeout to int error: %s", host, err.Error())
			return
		}
		var (
			auth         []ssh.AuthMethod
			addr         string
			clientConfig *ssh.ClientConfig
			client       *ssh.Client
			//session      *ssh.Session
			//err error
		)
		// get auth method
		auth = make([]ssh.AuthMethod, 0)
		auth = append(auth, ssh.Password(password))

		clientConfig = &ssh.ClientConfig{
			User:    user,
			Auth:    auth,
			Timeout: time.Duration(t) * time.Second,
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

// Release 内置命令，释放连接
func Release(host string) {
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

// Put 内置命令，批量上传文件
func Put(file string, dstDir string) {
	//fmt.Println("func put:", file, toDir)
	for host, client := range cliMap {
		wg.Add(1)
		go func(host string, client *Client, file, dstDir string) {
			defer wg.Done()
			toDir := dstDir
			if toDir == "" {
				toDir = client.HomePath
			}
			remoteFileName := path.Base(file)
			remotePath := path.Join(toDir, remoteFileName)
			session, err := client.Cli.NewSession()
			if err != nil {
				log.Errorf("[%s] get session error: %s", host, err.Error())
				return
			}
			err = scp.CopyPath(file, remotePath, session)
			if err != nil {
				log.Errorf("[%s] scp file %s error: %s", host, file, err)
				return
			}
			log.Infof("[%s] put file [%s] to [%s] success", host, file, remotePath)
		}(host, client, file, dstDir)
	}
	Done()
}

// Get 内置命令，批量下载文件
func Get(file string) {
	for host, client := range cliMap {
		wg.Add(1)
		go func(host string, client *Client, file string) {
			defer wg.Done()
			sftpClient, err := sftp.NewClient(client.Cli)
			if err != nil {
				log.Errorf("[%s] get sftpClient error: %s", host, err)
				return
			}
			defer sftpClient.Close()

			srcFile, err := sftpClient.Open(file)
			if err != nil {
				log.Errorf("[%s] open srcFile error: %s", host, err)
				return
			}
			defer srcFile.Close()

			localFileName := path.Base(file)
			localDir := path.Join(".", "download", host)
			os.MkdirAll(localDir, 0777)
			localPath := path.Join(localDir, localFileName)
			dstFile, err := os.Create(localPath)
			if err != nil {
				log.Errorf("[%s] create dstFile error: %s", host, err)
				return
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
		}(host, client, file)
	}
	Done()
}

// Check 内置命令，检测已建立连接
func Check() {
	for host, client := range cliMap {
		_ = client
		log.Infof("[%s] connecting", host)
	}
}

// Remote 内置命令，批量远程执行
func Remote(cmd string) {
	for host, client := range cliMap {
		fmt.Printf("\033[33m>>>>>>>>>>>> %s <<<<<<<<<<<<\033[0m\n", host)
		session, err := client.Cli.NewSession()
		//session.Stdin = os.Stdin
		session.Stdout = os.Stdout
		//session.Stderr = os.Stderr
		if err != nil {
			log.Errorf("[%s] get session error: %s\n", host, err.Error())
			continue
		}
		err = session.Run(cmd)
		if err != nil {
			// 此次执行有错误
			log.Errorf("[%s] remote command [%s] failed: %s\n", host, cmd, err.Error())
			continue
		}
		log.Infof("[%s] remote command [%s] success\n", host, cmd)
		session.Close()
	}
}

// Clear 内置命令，清屏
func Clear() {
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

// Vim 打开vim编辑器
func Vim(filename string) {
	if runtime.GOOS != "windows" {
		cmd := exec.Command("vim", filename)
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			log.Errorf("open vim failed, error: %s", err.Error())
			return
		}
		log.Infof("vim edit success")
	} else {
		log.Warnf("windows does not support vim")
	}
}

// Exit 退出mssh命令行
func Exit() {
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
