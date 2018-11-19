// 配置文件操作
// sdjdd @ 2018

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
)

type config struct {
	HotelCode string `json:"hotel_code"`
	MySQL     struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"mysql"`
	API struct {
		BaseURL string `json:"base_url"`
		Token   string `json:"token"`
	} `json:"api"`
	MaxPacketLength int    `json:"max_packet_length"`
	MaxSleepSecond  int    `json:"max_sleep_second"`
	BeginTime       string `json:"begin_time"`
}

// parseConfig 读取并解析配置文件
func parseConfig(filename string) (cfg *config, err error) {
	var data []byte
	if data, err = ioutil.ReadFile(filename); err != nil {
		return nil, errors.New("读取配置文件: " + err.Error())
	}
	cfg = new(config)
	if err = json.Unmarshal(data, cfg); err != nil {
		return nil, errors.New("解析配置文件: " + err.Error())
	}
	if !strings.HasSuffix(cfg.API.BaseURL, "/") {
		cfg.API.BaseURL += "/"
	}
	if cfg.MaxPacketLength <= 0 {
		return nil, errors.New(`配置参数 "max_packet_length" 必须大于0`)
	}
	return
}
