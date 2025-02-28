package httpcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/AsakuraMori/bangumi_go_api/pkg/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{}
var logger = types.InitBasicLogger("httpLog.log")

// 可以通过 ctx, cancel := context.WithTimeout(ctx, time.Millisecond) 设置超时
func Http(ctx context.Context, method, url string, params map[string]string, headers map[string]string,
	body []byte, retryCount uint) (int, string, error) {
	return HttpWithCli(ctx, httpClient, method, url, params, headers, body, retryCount)
}

func HttpJson(ctx context.Context, method, url string, params map[string]string, headers map[string]string,
	retryCount uint, in, out interface{}) error {
	return HttpJsonWithCli(ctx, httpClient, method, url, params, headers, retryCount, in, out)
}

func HttpWithCli(ctx context.Context, cli *http.Client, method, url string, params map[string]string,
	headers map[string]string, body []byte, retryCount uint) (int, string, error) {
	errMsg := ""
	for i := uint(0); i < retryCount+1; i += 1 {
		if i > 0 {
			d := time.Second * time.Duration(i)
			logger.Printf("[retry=%d] HttpRequestWithCli sleep %v", i, d)
			time.Sleep(d)
		}
		var bodyR io.Reader
		if len(body) > 0 {
			bodyR = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, url, bodyR)
		if err != nil {
			errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli url %s format error: %v, ", i, url, err)
			logger.Error(errMsg)
			continue
		}
		vals := req.URL.Query()
		for k, v := range params {
			vals.Add(k, v)
		}
		req.URL.RawQuery = vals.Encode()
		for k, v := range headers {
			req.Header.Add(k, v)
		}
		resp, err := cli.Do(req.WithContext(ctx))
		if err != nil {
			errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli %s , body: %s, error: %v, ", i, url, string(body), err)
			logger.Error(errMsg)
			continue
		}
		content, err := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli %s , body: %s, read body error: %v, ", i, url, string(body), err)
			logger.Error(errMsg)
			continue
		}
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout {
			if i < retryCount {
				errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli %s , body: %s, status_code: %v, ", i, url, string(body), resp.StatusCode)
				logger.Error(errMsg)
				continue
			}
		}
		return resp.StatusCode, string(content), nil
	}
	basicError := types.NewBasicError(errMsg)
	return 0, "", basicError
}

func HttpJsonWithCli(ctx context.Context, cli *http.Client, method, url string, params map[string]string,
	headers map[string]string, retryCount uint, in, out interface{}) error {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"

	statusCode, content, err := HttpWithCli(ctx, cli, method, url, params, headers, body, retryCount)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		err = types.NewBasicError(fmt.Sprintf(
			"http error, url: %s, request body: %s, status_code: %d, response body: %s",
			url, string(body), statusCode, content))
		logger.Error("%v", err)
		return err
	}
	if out != nil {
		if err := json.Unmarshal([]byte(content), out); err != nil {
			err = types.NewBasicError(fmt.Sprintf(
				"json.Unmarshal error, json string: %s", content))
			logger.Error("%v", err)
			return err
		}
	}
	return nil
}

func HttpFromUrlEncode(ctx context.Context, cli *http.Client, method, totalUrl string, params map[string]string,
	headers map[string]string, body map[string]string, retryCount uint) (int, string, error) {
	errMsg := ""
	for i := uint(0); i < retryCount+1; i += 1 {
		if i > 0 {
			d := time.Second * time.Duration(i)
			logger.Debug("[retry=%d] HttpRequestWithCli sleep %v", i, d)
			time.Sleep(d)
		}

		form := url.Values{}
		for k, v := range body {
			form.Add(k, v)
		}
		strings.NewReader(form.Encode())
		req, err := http.NewRequest(method, totalUrl, strings.NewReader(form.Encode()))

		if err != nil {
			errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli url %s format error: %v, ", i, totalUrl, err)
			logger.Error("%s", errMsg)
			continue
		}
		vals := req.URL.Query()
		for k, v := range params {
			vals.Add(k, v)
		}

		req.URL.RawQuery = vals.Encode()
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := cli.Do(req.WithContext(ctx))
		if err != nil {
			errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli %s , error: %v, ", i, totalUrl, err)
			logger.Error("%s", errMsg)
			continue
		}
		content, err := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli %s ,read body error: %v, ", i, totalUrl, err)
			logger.Error("%s", errMsg)
			continue
		}
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout {
			if i < retryCount {
				errMsg += fmt.Sprintf("[retry=%d] HttpRequestWithCli %s ,  status_code: %v, ", i, totalUrl, resp.StatusCode)
				logger.Error("%s", errMsg)
				continue
			}
		}
		return resp.StatusCode, string(content), nil
	}
	basicError := types.NewBasicError(errMsg)
	return 0, "", basicError
}

type RequestJson struct {
	Method  string
	Url     string
	Params  map[string]string
	Headers map[string]string
	Body    interface{}
	Retry   uint
	Timeout time.Duration
}

func HttpRequestJsonWithCli(ctx context.Context, cli *http.Client, req *RequestJson, out interface{}) error {
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	//logs.CtxInfo(ctx, "http request: %s", utils.Json(req))
	return HttpJsonWithCli(ctx, cli, req.Method, req.Url, req.Params, req.Headers, req.Retry, req.Body, out)
}
