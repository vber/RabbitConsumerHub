package utils

import (
	"fmt"
	"go-rabbitmq-consumers/types"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type HTTP_REQUEST_METHOD int

var (
	client *fasthttp.Client
)

const (
	HTTP_GET HTTP_REQUEST_METHOD = iota
	HTTP_POST
)

func init() {
	client = &fasthttp.Client{
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		// MaxIdleConnDuration: 1 * time.Hour,
	}
}

func GetUUID() string {
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)
	return uuid
}

func HttpRequest(method HTTP_REQUEST_METHOD, headers map[string]string, url, body string) (string, error, int) {
	var (
		httpMethod string
		err        error
		statusCode int
	)
	if method == HTTP_GET {
		httpMethod = "GET"
	} else {
		httpMethod = "POST"
	}

	req := fasthttp.AcquireRequest()

	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(httpMethod)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("vinehoo-client", types.HEADER_VINEHOO_CLIENT)
	req.Header.Add("vinehoo-client-version", types.HEADER_CLIENT_VERSION)

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	req.SetRequestURI(url)
	if method == HTTP_POST {
		req.SetBodyString(body)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = client.Do(req, resp)
	if err != nil {
		return "", err, 0
	}

	statusCode = resp.StatusCode()
	if statusCode != 200 {
		return "", fmt.Errorf("http status code: %d", statusCode), statusCode
	}

	return string(resp.Body()), nil, statusCode
}
