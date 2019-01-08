package main

import (
	"flag"
	"github.com/peeped/cgp/utils"
	"log"
	"os"
)

var (
	workspace = os.Getenv("BeeWorkspace")
)

func main() {
	currentpath, _ := os.Getwd()
	if workspace != "" {
		currentpath = workspace
	}

	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()

	if len(args) < 1 {
		log.Println("请输入要创建的项目名称")
		os.Exit(2)
	}

	if args[0] == "help" {
		log.Println("本项目没有帮助信息")
		os.Exit(2)
	}

	if args[0] == "run" {
		log.Println("本项目还不支持调试运行")
		os.Exit(2)
	}

	if args[0] == "new" && args[1] != "" {
		if utils.IsInGOPATH(currentpath) {
			utils.CreateProject(currentpath, args[1])
		} else {
			log.Println("请将项目创建在GOPATH中")
			os.Exit(2)
		}
	}

}
