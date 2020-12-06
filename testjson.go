package main

import (
	"fmt"
	"github.com/gitcfly/ngredt/config"
)

func main() {
	fileConf := "./ngredt_client.conf"
	if config.InitConf(fileConf) != nil {
		return
	}
	fmt.Println(config.NgConf)
}
