// 对数据库操作的封装
// sdjdd @ 2018

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type database struct {
	mysql *sql.DB
}

type operationData [5]interface{}

// newMysql 返回一个包含 Mysql 连接的 database 指针
func newMysql(host string, port int, user, password, dbname string) (db *database, err error) {
	db = new(database)
	constr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbname)
	db.mysql, err = sql.Open("mysql", constr)
	if err != nil {
		return nil, errors.New("打开数据库: " + err.Error())
	}
	return
}

func (db *database) testConnection() error {
	if err := db.mysql.Ping(); err != nil {
		return errors.New("连接数据库: " + err.Error())
	}
	return nil
}

func (db *database) close() error {
	return db.mysql.Close()
}

func (db *database) getOperationData(beginID, length int, beginDate string) (data []operationData, err error) {
	var rows *sql.Rows
	var timecdt string
	if beginID == 1 {
		// 同步第一条数据是加上 time 条件约束
		timecdt = " AND time>='" + beginDate + "'"
	}
	sql := "SELECT operate_id,function_name,time,status,stb_ip FROM operate" +
		" WHERE operate_id>=%d AND del_flag='0'" + timecdt +
		" ORDER BY operate_id LIMIT %d;"
	sql = fmt.Sprintf(sql, beginID, length)
	if rows, err = db.mysql.Query(sql); err != nil {
		return nil, errors.New("执行数据库查询: " + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var row operationData
		var operateID int
		var functionName, time, status, stbIP string
		rows.Scan(&operateID, &functionName, &time, &status, &stbIP)
		row[0] = operateID
		row[1] = functionName
		row[2] = time[:8] + time[9:] // 去掉 time 中间的空格, "20060102 150405" -> "20060102150405"
		if strings.EqualFold(status, "IN") {
			row[3] = 1
		} else {
			row[3] = 0
		}
		row[4] = stbIP
		data = append(data, row)
	}
	if rows.Err() != nil {
		return nil, errors.New("执行查询意外结束: " + err.Error())
	}
	return data, nil
}
