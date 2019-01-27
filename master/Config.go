package master

import (
	"github.com/segmentio/objconv/json"
	"io/ioutil"
)
var (
	G_config *Config
)
type Config struct{
	ApiPort int `json:"apiPort"`
	ApiReadTimeout int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`
	EtcdEndpoints []string `json:"etcdEndpoints"`
	EtcdDialTimeout int `json:"etcdDialTimeout"`
}
//加载配置
func InitConfig(filename string)(err error){
	var (
		content []byte
		conf Config
	)
	if content,err=ioutil.ReadFile(filename);err!=nil{
		return
	}
	if err=json.Unmarshal(content,&conf);err!=nil{
		return
	}
	G_config=&conf
	return
}