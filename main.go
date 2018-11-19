package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"time"
)

var (
	cfg         *config     // 配置文件
	db          *database   // 数据库连接池
	myAPIHelper *apiHelper  // API 调用工具
	linfWriter  = os.Stdout // 信息日志 Writer
	lerrWriter  = os.Stderr // 错位日志 Writer
	linf        = log.New(linfWriter, "[信息] ", log.LstdFlags)
	lwar        = log.New(linfWriter, "[警告] ", log.LstdFlags)
	lerr        = log.New(lerrWriter, "[错误] ", log.LstdFlags)
)

const (
	configFilename = "config.json" // 配置文件名
)

func main() {
	var err error
	if cfg, err = parseConfig(configFilename); err != nil {
		lerr.Fatalln("启动失败: " + err.Error())
	}
	if db, err = newMysql(cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Username, cfg.MySQL.Password, cfg.MySQL.Database); err != nil {
		lerr.Fatalln("启动失败: " + err.Error())
	}
	defer db.close()
	if err = db.testConnection(); err != nil {
		lerr.Fatalln("启动失败: " + err.Error())
	}
	linf.Println("程序启动")
	rand.Seed(time.Now().Unix()) // 设置随机数种子
	myAPIHelper = newAPIHelper(cfg.API.BaseURL, cfg.API.Token, cfg.HotelCode)

	for {
		var lastID int
		var data []operationData
		lastID, err = myAPIHelper.getLastOperationID()
		if err != nil {
			if err == errHotelNotExists || err == errSyncNotEnabled {
				lwar.Println("数据库同步已推迟: " + err.Error())
			} else {
				lerr.Println("获取 last_operation_id 失败: " + err.Error())
			}
			sleepOnError(5)
			continue
		}
		data, err = db.getOperationData(lastID+1, cfg.MaxPacketLength, cfg.BeginTime)
		if err != nil {
			lerr.Println("获取 operation data 失败: " + err.Error())
			sleepOnError(5)
			continue
		}
		dataLength := len(data)
		if dataLength == 0 {
			linf.Println("暂无新数据")
			time.Sleep(time.Minute)
			continue
		}
		jsonData, _ := json.Marshal(data)
		err = myAPIHelper.postOperationData(jsonData)
		if err != nil {
			lerr.Println("上传数据失败: " + err.Error())
		}
		linf.Printf("成功上传数据, 数据量: %d, id: [%d, %d]\n", dataLength, data[0][0], data[dataLength-1][0])
		sleep()
	}
}

// sleep 根据配置文件中的设定随机等待一段时间
func sleep() {
	if cfg.MaxSleepSecond > 0 {
		duration := time.Duration(rand.Intn(cfg.MaxSleepSecond)+1) * time.Second
		time.Sleep(duration)
	}
}

func sleepOnError(minute int) {
	linf.Printf("%d分钟后重试\n", minute)
	time.Sleep(time.Minute * time.Duration(minute))
}
