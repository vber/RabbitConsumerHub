package retry

import (
	"encoding/json"
	"fmt"
	"go-rabbitmq-consumers/logger"
	"go-rabbitmq-consumers/types"
	"go-rabbitmq-consumers/utils"
	"strconv"
	"strings"
	"time"
)

// 文档: https://showdoc.wineyun.com/web/#/68/2710

type RetryURL struct {
	QueueData string  // 已debase64的队列数据
	ReqURL    string  // 请求接口地址
	RetryMode string  // 重试机制
	RetryAPI  *string // 重试服务器接口
}

type APIReturn struct {
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

func parseRetryMode(retryMode string) string {
	var (
		arrNewRetryMode []string
	)
	arrRetryDuration := strings.Split(retryMode, ",")
	arrNewRetryMode = make([]string, len(arrRetryDuration))
	for i, r := range arrRetryDuration {
		if r2, err := time.ParseDuration(r); err != nil {
			logger.E("parseRetryMode:", err, retryMode)
			return ""
		} else {
			arrNewRetryMode[i] = strconv.Itoa(int(r2.Seconds()))
		}
	}
	return strings.Join(arrNewRetryMode, ",")
}

func (r *RetryURL) RetryRequest() error {
	var (
		err        error
		body       string
		api_return APIReturn
	)

	body, err, _ = utils.HttpRequest(utils.HTTP_POST, map[string]string{
		"vinehoo-client":         types.HEADER_VINEHOO_CLIENT,
		"vinehoo-client-version": types.HEADER_CLIENT_VERSION,
		"RetryValue":             r.RetryMode,
		"RetryUrl":               r.ReqURL,
		"RetryId":                utils.GetUUID(),
	}, fmt.Sprintf("%s/retryapi", *r.RetryAPI), r.QueueData)

	fmt.Println("retry request:", r.ReqURL, *r.RetryAPI, r.QueueData, ",retryvalue:", r.RetryMode)
	if err != nil {
		logger.E("Retry", err.Error())
		return err
	} else {
		logger.I("Retry", fmt.Sprintf("%s return:%s", r.ReqURL, body))
		err = json.Unmarshal([]byte(body), &api_return)
		if err != nil {
			return fmt.Errorf("retry error:%s", err.Error())
		} else {
			if api_return.ErrorCode == 0 {
				return nil
			} else {
				return fmt.Errorf("retry error:%s", api_return.ErrorMsg)
			}
		}
	}
}
