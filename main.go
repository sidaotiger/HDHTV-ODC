package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type config struct {
	HotelID int
	MySQL   struct {
		Host     string
		Port     int
		User     string
		Password string
		Database string
	}
	Upload struct {
		URL      string
		FormName string
	}
}

type operateData struct {
	HotelID int
	Time    string
	Data    [][]string
}

const (
	dumpTimeFile = "next-dump-time"
)

var (
	cfg     config // 配置数据
	lerr    = log.New(os.Stderr, "[错误] ", log.LstdFlags)
	linf    = log.New(os.Stdout, "[信息] ", log.LstdFlags)
	dumping bool
)

func main() {
	var err error
	cfg, err = parseConfig("config.json") // 解析配置文件
	if err != nil {
		lerr.Fatalln(err)
	}
	linf.Println("程序启动")
	tkch := time.Tick(time.Minute)
	for now := range tkch {
		if !dumping {
			go dump(now)
		}
	}
}

func dump(now time.Time) {
	dumping = true
	curHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	nextTime, err := getNextDumpTime()
	if err != nil {
		lerr.Println(err)
	} else {
		for nextTime.Before(curHour) {
			linf.Println("收集数据: " + nextTime.Format("2006-01-02 15:04:05"))
			data, err := getOperationData(nextTime)
			if err != nil {
				lerr.Println("收集数据失败: " + err.Error())
				break
			}
			if err = writeData(data); err != nil {
				lerr.Println("数据写入失败: " + err.Error())
				break
			}
			nextTime = nextTime.Add(time.Hour)
			if err = setNextDumpTime(nextTime); err != nil {
				lerr.Println("更新时间失败: " + err.Error())
				break
			}
		}
	}
	uploadAllFiles()
	dumping = false
}

func uploadAllFiles() {
	files, err := ioutil.ReadDir("data")
	if err != nil {
		lerr.Println("读取data目录失败: " + err.Error())
		return
	}
	for _, file := range files {
		filename := "data/" + file.Name()
		if err := postFile(filename, cfg.Upload.URL, cfg.Upload.FormName); err != nil {
			lerr.Println("上传文件失败: " + err.Error())
			break
		}
		linf.Println("上传文件: " + filename)
		os.Remove(filename)
	}
}

// 读取并解析配置文件
func parseConfig(filename string) (config, error) {
	var cfg config
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return cfg, errors.New("读取配置文件失败: " + err.Error())
	}
	if err = json.Unmarshal(data, &cfg); err != nil {
		return cfg, errors.New("解析配置文件失败: " + err.Error())
	}
	return cfg, nil
}

// 读取下次收集的时间
func getNextDumpTime() (time.Time, error) {
	var t time.Time
	data, err := ioutil.ReadFile(dumpTimeFile)
	if err != nil {
		return t, errors.New("读取时间失败: " + err.Error())
	}
	if t, err = time.Parse("2006010215", string(data)); err != nil {
		return t, errors.New("解析时间失败: " + dumpTimeFile + "格式不正确")
	}
	return t, nil
}

// 将下次收集日期写入文件
func setNextDumpTime(t time.Time) error {
	return ioutil.WriteFile(dumpTimeFile, []byte(t.Format("2006010215")), 0666)
}

// 读取指定时间的数据
func getOperationData(t time.Time) (*operateData, error) {
	timeStr := t.Format("20060102 15")
	data := operateData{
		HotelID: cfg.HotelID,
		Time:    t.Format("2006010215"),
		Data:    [][]string{},
	}
	// 数据库连接字符串
	conStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Database)
	db, err := sql.Open("mysql", conStr)
	if err != nil {
		return nil, errors.New("打开数据库失败: " + err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return nil, errors.New("连接数据库失败: " + err.Error())
	}
	sql := fmt.Sprintf(`SELECT function_name,time,status,stb_ip FROM operate 
		WHERE time BETWEEN '%[1]s0000' AND '%[1]s5959' AND del_flag='0' ORDER BY operate_id`, timeStr)
	rows, err := db.Query(sql)
	if err != nil {
		return nil, errors.New("执行数据库查询失败: " + err.Error())
	}
	for rows.Next() {
		temp := make([]string, 4)
		rows.Scan(&temp[0], &temp[1], &temp[2], &temp[3])
		data.Data = append(data.Data, temp)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("执行数据库查询时发生错误: " + err.Error())
	}
	return &data, nil
}

func writeData(data *operateData) error {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}
	/* if d, err = GzipEncode(d); err != nil {
		return errors.New("压缩数据失败: " + err.Error())
	} */
	filename := fmt.Sprintf("%d@%s.json", cfg.HotelID, data.Time)
	os.Mkdir("data", 0755)
	return ioutil.WriteFile("data/"+filename, d, 0644)
}

func postFile(filename, targetURL, formName string) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile(formName, filename)
	if err != nil {
		return err
	}
	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	resp, err := http.Post(targetURL, contentType, bodyBuf)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("服务器错误")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" {
		return errors.New("服务器错误")
	}
	return nil
}

// GzipEncode compress bytes by gzip
func GzipEncode(in []byte) ([]byte, error) {
	var (
		buffer bytes.Buffer
		out    []byte
		err    error
	)
	writer := gzip.NewWriter(&buffer)
	_, err = writer.Write(in)
	if err != nil {
		writer.Close()
		return out, err
	}
	err = writer.Close()
	if err != nil {
		return out, err
	}

	return buffer.Bytes(), nil
}

// GzipDecode uncompress bytes by gzip
func GzipDecode(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		var out []byte
		return out, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}
