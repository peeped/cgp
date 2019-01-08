package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var appconf = `[sysInfo]
app_name = {{.Appname}}
run_port = :7070
run_mode = debug

[mysqlInfo]
dbname = databasename
dbuser = root
dbpwd = password
dbhost = localhost:3306

[redisInfo]
redis_Host = localhost:6379
redis_IsAuth = 0
redis_Auth = aaa:bbb
redis_MaxIdle = 20
redis_MaxActive = 1000
redis_Prefix = prefix
redis_Select = 9

`
var controller = `package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"{{.Appname}}/model"
)

func AuctionController(c *gin.Context) {
	realse := model.NewRealse()

	c.JSON(http.StatusOK, realse)
	return
}
`

var res = `package model

import (
	"fmt"
	"github.com/astaxie/beego/logs"
)

var (
	MysqlInfo = make(map[string]string) // MYSQL配置信息
	RedisInfo = make(map[string]string) // Redis配置信息
	SysInfo   = make(map[string]string) // 程序运行端口
	Log       *logs.BeeLogger
)

//响应码
const (
	CodeStatusOK CodeStatus = 0  // 成功
)

// 响应信息
var messageTmpls = map[string]string{
	CodeStatusOK.String(): "成功",
}

//响应码
type CodeStatus int16
type ReportStatus string

// 返回结构
func NewRealse() *Realse {
	return &Realse{"0", "0", make(map[string]interface{})}
}

type Realse struct {
	Ret  string                 {{.DOT}}json:"ret"{{.DOT}}
	Msg  string                 {{.DOT}}json:"msg"{{.DOT}}
	Data map[string]interface{} {{.DOT}}json:"data"{{.DOT}}
}

func (self CodeStatus) String() string {
	return fmt.Sprintf("%d", self)
}

func (self CodeStatus) Message() string {
	return messageTmpls[self.String()]
}

func (self *Realse) Write(code CodeStatus) {
	self.Ret = code.String()
	self.Msg = code.Message()
}

`

var service = `package service

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"{{.Appname}}/model"
	"log"
	"time"
)

var (
	MYSQLPOOL *xorm.Engine // 数据库连接池
	REDISPOOL *redis.Pool  // 数据库连接池
)

//初始化MYSQL
func InitMysql() {
	mysqlDns := fmt.Sprintf("%s:%s@tcp(%s)/%s?timeout=3s&parseTime=true&loc=Local&charset=utf8", model.MysqlInfo["dbuser"], model.MysqlInfo["dbpwd"], model.MysqlInfo["dbhost"], model.MysqlInfo["dbname"])
	if conn, err := xorm.NewEngine("mysql", mysqlDns); err != nil {
		log.Fatal("mysql db connection err", err)
	} else {
		MYSQLPOOL = conn
	}
	MYSQLPOOL.ShowSQL(true)
}

//初始化REDIS
func InitRedis() {
	REDISPOOL = &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", model.RedisInfo["redis_Host"])
			if err != nil {
				log.Fatal("Init Redis Failed:", err.Error())
				return nil, err
			}
			if model.RedisInfo["redis_IsAuth"] == "1" {
				if _, err := c.Do("AUTH", model.RedisInfo["redis_Auth"]); err != nil {
					log.Fatal("Init Redis Failed:", err.Error())
					c.Close()
					return nil, err
				}
			}
			if _, err = c.Do("SELECT", model.RedisInfo["redis_Select"]); err != nil {
				log.Fatal("Init Redis Failed:", err.Error())
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}
}
`

var router = `package routers

import (
	"github.com/gin-gonic/gin"
	"{{.Appname}}/controller"
)

func Route(engine *gin.Engine) {
	// Examples 
	// engine.GET("/api/v1/auction", controller.AuctionController)
	// engine.POST("/api/v1/auction", controller.AddAuctionController)
	// engine.PUT("/api/v1/somePut", controller.PuttingController)
	// engine.DELETE("/api/v1/someDelete", controller.DeletingController)
	// engine.PATCH("/api/v1/somePatch", controller.PatchingController)
	// engine.HEAD("/api/v1/someHead", controller.HeadController)
	// engine.OPTIONS("/api/v1/someOptions", controller.OptionsController)

	// Examples route group
	// v2 := engine.Group("/api/v2")
	// {
		// v2.GET("/", controller.groupController)
		// v2.GET("/g1", controller.group1Controller)
		// v2.GET("/g2", controller.group2Controller)
	// }

	engine.GET("/api/v1/auction", controller.AuctionController)
}
`

var config = `package main

import (
	"flag"
	"github.com/astaxie/beego/logs"
	"github.com/larspensjo/config"
	"{{.Appname}}/model"
	"{{.Appname}}/service"
	"log"
)

var configFile string

func init() {
	model.Log = logs.NewLogger(1000)
	model.Log.SetLogger("file", {{.DOT}}{"filename":"logs/run.log"}{{.DOT}})
	model.Log.EnableFuncCallDepth(true)

	flag.StringVar(&configFile, "c", "./conf/app.ini", "General configuration file")
	flag.Parse()

	model.Log.Info("==============================")
	model.Log.Info("= 初始化配置文件 ...")
	initConf()
	model.Log.Info("= 初始化mysql连接池 ...")
	service.InitMysql()
	model.Log.Info("= 初始化redis连接池 ...")
	service.InitRedis()
}

// 初始化配置文件
func initConf() {
	//=====set config file std
	conf, err := config.ReadDefault(configFile)
	if err != nil {
		log.Fatal("Fail to find", configFile, err)
	}

	model.Log.Info("=【初始化配置mysql】")
	//read mysql conf
	l, err := conf.Options("mysqlInfo")
	if err != nil {
		log.Fatal("Fail to find mysql config ", err)
	}
	for _, k := range l {
		v, err := conf.String("mysqlInfo", k)
		if err != nil {
			log.Fatalf("mysqlInfo [%s] err:%v", k, err)
		}
		model.Log.Info("=	%s:%s", k, v)
		model.MysqlInfo[k] = v
	}

	model.Log.Info("=【初始化配置sysinfo】")
	//read run port conf
	l, err = conf.Options("sysInfo")
	if err != nil {
		log.Fatal("Fail to find sys Info config ", err)
	}
	for _, k := range l {
		v, err := conf.String("sysInfo", k)
		if err != nil {
			log.Fatalf("sys Info [%s] err:%v", k, err)
		}
		model.Log.Info("=	%s:%s", k, v)
		model.SysInfo[k] = v
	}

	model.Log.Info("=【初始化配置redis】")
	//read redis conf
	l, err = conf.Options("redisInfo")
	if err != nil {
		log.Fatal("Fail to find redis Info config ", err)
	}
	for _, k := range l {
		v, err := conf.String("redisInfo", k)
		if err != nil {
			log.Fatalf("redis Info [%s] err:%v", k, err)
		}
		model.Log.Info("=	%s:%s", k, v)
		model.RedisInfo[k] = v
	}
	model.Log.Info("==============================")
}

`

var maingo = `package main

import (
	"github.com/gin-gonic/gin"
	"{{.Appname}}/model"
	"{{.Appname}}/routers"
)

func main() {
	r := gin.New()
	routers.Route(r)
	r.Static("/static", "./static")
	gin.SetMode(model.SysInfo["run_mode"])
	r.Run(model.SysInfo["run_port"])
}
`

func CreateProject(path, name string) {
	apppath := filepath.Join(path, name)

	os.MkdirAll(apppath, 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", apppath+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "conf"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "conf")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "logs"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "logs")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "controller"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "controller")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "model"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "model")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "routers"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "routers")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "tests"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "tests")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "static"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "static")+string(filepath.Separator), "\x1b[0m\n")

	os.Mkdir(filepath.Join(apppath, "service"), 0755)
	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "service")+string(filepath.Separator), "\x1b[0m\n")

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "conf")+string(filepath.Separator), "\x1b[31mapp.ini\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "conf", "app.ini"), strings.Replace(appconf, "{{.Appname}}", filepath.Base(name), -1))

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "controller")+string(filepath.Separator), "\x1b[31mdefault.go\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "controller", "default.go"), strings.Replace(controller, "{{.Appname}}", name, -1))

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "model")+string(filepath.Separator), "\x1b[31mres.go\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "model", "res.go"), strings.Replace(res, "{{.DOT}}", "`", -1))

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "routers")+string(filepath.Separator), "\x1b[31mrouter.go\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "service", "service.go"), strings.Replace(service, "{{.Appname}}", name, -1))

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath, "routers")+string(filepath.Separator), "\x1b[31mrouter.go\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "routers", "router.go"), strings.Replace(router, "{{.Appname}}", name, -1))

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath)+string(filepath.Separator), "\x1b[31mconfig.go\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "config.go"), strings.Replace(strings.Replace(config, "{{.Appname}}", name, -1), "{{.DOT}}", "`", -1))

	fmt.Print("\t\x1b[32mCreate\t", "\x1b[1m", filepath.Join(apppath)+string(filepath.Separator), "\x1b[31mmain.go\x1b[0m\n")
	WriteToFile(filepath.Join(apppath, "main.go"), strings.Replace(maingo, "{{.Appname}}", name, -1))

}
