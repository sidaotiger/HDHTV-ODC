// 封装 API 调用
// sdjdd @ 2018

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type apiHelper struct {
	token                string
	urlGetOperationData  string
	urlPostOperationData string
}

type apiClient struct {
	client http.Client
}

type responseData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

var (
	errHotelNotExists = errors.New("酒店不存在, 请检查 hotel_code 是否有误")
	errSyncNotEnabled = errors.New("当前酒店未开启数据库同步功能, 请联系管理员开启")
	errInvalidData    = errors.New("数据格式不正确, 接口可能被修改, 请联系管理员修复")
	errInsertFailed   = errors.New("插入数据失败, 请确保只运行了一个程序进程")
)

func newAPIHelper(baseURL, token, hotelCode string) *apiHelper {
	helper := apiHelper{
		token:                token,
		urlGetOperationData:  baseURL + hotelCode + "/last_id",
		urlPostOperationData: baseURL + hotelCode,
	}
	return &helper
}

// getLastOperationID 获取 last_operation_id
func (h *apiHelper) getLastOperationID() (int, error) {
	var client apiClient
	req, _ := http.NewRequest(http.MethodGet, h.urlGetOperationData, nil)
	req.Header.Set("token", h.token)
	data, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	if data.Code != 0 {
		switch data.Code {
		case 1:
			return 0, errHotelNotExists
		case 2:
			return 0, errSyncNotEnabled
		default:
			return 0, errors.New("意料之外的错误: " + data.Msg)
		}
	}
	if lastOperationID, ok := data.Data.(float64); ok {
		return int(lastOperationID), nil
	}
	return 0, errors.New("返回值非法, 接口可能被修改, 请联系管理员修复")
}

func (h *apiHelper) postOperationData(data []byte) error {
	var client apiClient
	req, _ := http.NewRequest(http.MethodPost, h.urlPostOperationData, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", h.token)
	respData, err := client.Do(req)
	if err != nil {
		return err
	}
	if respData.Code != 0 {
		switch respData.Code {
		case 1:
			return errHotelNotExists
		case 2:
			return errSyncNotEnabled
		case 3:
			return errInvalidData
		default:
			return errors.New("意料之外的错误: " + respData.Msg)
		}
	}
	return nil
}

func (c *apiClient) Do(request *http.Request) (*responseData, error) {
	resp, err := c.client.Do(request)
	if err != nil {
		return nil, errors.New("发送 HTTP 请求: " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP status: " + resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("读取 response body: " + err.Error())
	}
	data := new(responseData)
	if err = json.Unmarshal(body, data); err != nil {
		return nil, errors.New("解析 response body: " + err.Error())
	}
	return data, nil
}
