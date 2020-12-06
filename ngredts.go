package main

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gitcfly/ngredt/config"
	"github.com/gitcfly/ngredt/ioutils"
	"github.com/gitcfly/ngredt/log"
	"github.com/satori/go.uuid"
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

var TcpRecords = list.New()

type TcpRecord struct {
	clientKey   string
	sgnConn     net.Conn
	tcpListener net.Listener
}

var privateKey2Server = map[string]net.Listener{}

var Request2Conn = &sync.Map{}

// tcp内网端口代理,支持所有以tcp为基础的协议(如tcp,http)，服务端实现
func main() {
	fileConf := "./ngredt_sever.conf"
	if len(os.Args) >= 2 {
		fileConf = os.Args[1]
	}
	if config.InitConf(fileConf) != nil {
		return
	}
	sgnAddr := fmt.Sprintf(":%v", config.NgConf.SgnPort)
	signalListener, err := net.Listen("tcp", sgnAddr)
	if err != nil {
		logrus.Error(err)
		return
	}
	go TcpConnPool()
	go HeartBreakCheck()
	logrus.Infof("服务启动。。，信令端口=%v,连接池端口=%v", config.NgConf.SgnPort, config.NgConf.ConnPort)
	for {
		signalConn, err := signalListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		go SignalClient(signalConn)
	}
}

func SignalClient(signalConn net.Conn) {
	privateBytes, _ := bufio.NewReader(signalConn).ReadBytes('\n')
	privateKey := string(bytes.TrimRight(privateBytes, "\n"))
	if listener := privateKey2Server[privateKey]; listener != nil {
		if err := listener.Close(); err != nil {
			logrus.Error(err)
		}
		delete(privateKey2Server, privateKey)
	}
	if len(config.NgConf.PortMap) == 0 {
		logrus.Error("未配置客户端口，key=%v", privateKey)
		return
	}
	port, ok := config.NgConf.PortMap[privateKey]
	if !ok || port == 0 {
		logrus.Error("未配置客户端口，key=%v", privateKey)
		return
	}
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%v", port)) //外部请求地址
	if err != nil {
		logrus.Error(err)
		return
	}
	TcpRecords.PushBack(&TcpRecord{
		clientKey:   privateKey,
		sgnConn:     signalConn,
		tcpListener: tcpListener},
	)
	signalConn.Write([]byte(fmt.Sprint(port) + "\n"))
	privateKey2Server[privateKey] = tcpListener
	logrus.Infof("为新客户端启动tcp转发。。，client_key=%v,服务端口=%v", privateKey, port)
	TcpServer(tcpListener, signalConn)
}

func HeartBreakCheck() {
	tmpData := make([]byte, 1)
	for range time.Tick(5 * time.Second) {
		var next *list.Element
		for e := TcpRecords.Front(); e != nil; e = next {
			next = e.Next()
			tcpRecord := e.Value.(*TcpRecord)
			if _, err := tcpRecord.sgnConn.Read(tmpData); err != nil {
				logrus.Infof("client_key=%v,客户端代理连接超时，服务端主动关闭连接以及tcp服务,err=%v", tcpRecord.clientKey, err)
				if err := tcpRecord.sgnConn.Close(); err != nil {
					logrus.Error(err)
				}
				if err := tcpRecord.tcpListener.Close(); err != nil {
					logrus.Error(err)
				}
				Request2Conn.Delete(tcpRecord.clientKey)
				TcpRecords.Remove(e)
			}
		}
	}
}

func TcpConnPool() {
	connAddr := fmt.Sprintf(":%v", config.NgConf.ConnPort)
	poolListener, err := net.Listen("tcp", connAddr) //内部连接池端口
	if err != nil {
		logrus.Error(err)
		return
	}
	for {
		tcpConn, err := poolListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		requestId, err := bufio.NewReader(tcpConn).ReadBytes('\n')
		Request2Conn.Store(string(requestId), tcpConn)
	}
}

func TcpServer(tcpListener net.Listener, signalConn net.Conn) {
	for {
		reqConn, err := tcpListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		requestId := uuid.NewV4().String() + "\n"
		logrus.Infof("写入requestId=%v", requestId)
		_, err = signalConn.Write([]byte(requestId)) //告知客户端需要主动发起连接请求
		if err != nil {
			logrus.Error(err)
			return
		}
		go RedirectConn(reqConn, requestId)
	}
}

func RedirectConn(reqConn net.Conn, requestId string) {
	defer func() {
		Request2Conn.Delete(requestId)
		logrus.Infof("请求处理结束，requestId=%v", requestId)
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()
	for {
		if tpConn, ok := Request2Conn.Load(requestId); ok {
			proxyConn := tpConn.(net.Conn)
			go ioutils.CopyTcp(reqConn, proxyConn)
			ioutils.CopyTcp(proxyConn, reqConn)
			break
		}
	}
}
