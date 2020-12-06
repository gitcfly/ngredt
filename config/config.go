package config

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var (
	NgConf *NgRedtConf
)

func InitConf(path string) error {
	confbytes, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Error(err)
		return err
	}
	err = json.Unmarshal(confbytes, &NgConf)
	if err != nil {
		logrus.Error(err)
	}
	return err
}

type NgRedtConf struct {
	SevHost    string         `json:"sev_host,omitempty"`    //服务地址,服务端用
	SgnPort    int            `json:"sgn_port,omitempty"`    //信令端口,双端用
	PortMap    map[string]int `json:"port_map,omitempty"`    //端口映射,服务端用
	ConnPort   int            `json:"conn_port,omitempty"`   //链接端口，双端用
	LocAddr    string         `json:"loc_addr,omitempty"`    //本机服务地址，host:port信息,一般为127.0.0.1:port
	PrivateKey string         `json:"private_key,omitempty"` //客户端key,客户端用
}
