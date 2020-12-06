package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gitcfly/ngredt/config"
	"github.com/gitcfly/ngredt/ioutils"
	"github.com/gitcfly/ngredt/log"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		TimestampFormat:        "2006-01-02 15:04:05",
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	logrus.AddHook(log.NewContextHook())
}

// tcp内网端口代理,支持所有以tcp为基础的协议(如tcp,http)，服务端实现
func main() {
	fileConf := "./ngredt_client.json"
	if len(os.Args) >= 2 {
		fileConf = os.Args[1]
	}
	if config.InitConf(fileConf) != nil {
		return
	}
	sgnAddr := fmt.Sprintf("%v:%v", config.NgConf.SevHost, config.NgConf.SgnPort)
	signalConn, err := net.Dial("tcp", sgnAddr)
	if err != nil {
		logrus.Error(err)
		return
	}
	signalConn.Write([]byte(config.NgConf.PrivateKey + "\n"))
	go func() {
		for range time.Tick(4 * time.Second) {
			signalConn.Write([]byte("y"))
		}
	}()
	sevPort := make([]byte, 20)
	n, _ := signalConn.Read(sevPort)
	pubPort := strings.TrimRight(string(sevPort[:n]), "\n")
	logrus.Infof("连接成功，本地服务地址=%v,外网服务地址=%v:%v", config.NgConf.LocAddr, config.NgConf.SevHost, pubPort)
	for {
		requestId, err := bufio.NewReader(signalConn).ReadBytes('\n')
		if err != nil {
			logrus.Error(err)
			signalConn = RetrySignalConn()
			continue
		}
		logrus.Infof("读取到requestId=%v", string(requestId))
		go HandleTcpConn(string(requestId))
	}
}

func RetrySignalConn() net.Conn {
	sgnAddr := fmt.Sprintf("%v:%v", config.NgConf.SevHost, config.NgConf.SgnPort)
	for range time.Tick(2 * time.Second) {
		if signalConn, err := net.Dial("tcp", sgnAddr); err == nil {
			signalConn.Write([]byte(config.NgConf.PrivateKey + "\n"))
			return signalConn
		} else {
			logrus.Error(err)
		}
	}
	return nil
}

func HandleTcpConn(requestId string) {
	defer func() {
		logrus.Infof("请求处理结束，requestId=%v", requestId)
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()
	connAddr := fmt.Sprintf("%v:%v", config.NgConf.SevHost, config.NgConf.ConnPort)
	proxyConn, err := net.Dial("tcp", connAddr)
	if err != nil {
		logrus.Error(err)
		return
	}
	proxyConn.Write([]byte(requestId))
	realConn, err := net.Dial("tcp", config.NgConf.LocAddr)
	if err != nil {
		logrus.Error(err)
		return
	}
	go ioutils.CopyTcp(proxyConn, realConn)
	ioutils.CopyTcp(realConn, proxyConn)
}
