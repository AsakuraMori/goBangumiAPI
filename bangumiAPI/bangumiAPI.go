package bangumiAPI

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"goBangumiAPI/bangumiAPI/httpcli"
	"net/http"
	"strconv"
	"strings"
)

type Client struct {
	Caller string

	host            string
	proxyUrl        string
	cli             *http.Client
	concurrencyChan chan bool
	skip            bool
}
type SubjectType int

const (
	SubjectTypeBook  = SubjectType(1)
	SubjectTypeAnime = SubjectType(2)
	SubjectTypeMusic = SubjectType(3)
	SubjectTypeGame  = SubjectType(4)
	SubjectTypeReal  = SubjectType(6)
)

func (st SubjectType) IsSupported() bool {
	return st == SubjectTypeBook || st == SubjectTypeAnime || st == SubjectTypeMusic || st == SubjectTypeGame || st == SubjectTypeReal
}

func (st SubjectType) ToString() string {
	switch st {
	case SubjectTypeBook:
		return "1"
	case SubjectTypeAnime:
		return "2"
	case SubjectTypeMusic:
		return "3"
	case SubjectTypeGame:
		return "4"
	case SubjectTypeReal:
		return "6"
	default:
		return ""
	}
}

func (st SubjectType) Name() string {
	switch st {
	case SubjectTypeBook:
		return "书籍"
	case SubjectTypeAnime:
		return "动画"
	case SubjectTypeMusic:
		return "音乐"
	case SubjectTypeGame:
		return "游戏"
	case SubjectTypeReal:
		return "三次元"
	default:
		return ""
	}
}

type respType struct {
	Total  int64           `json:"total"`
	Limit  int64           `json:"limit"`
	Offset int64           `json:"offset"`
	Data   json.RawMessage `json:"data"`
}

func NewClient(caller string, host string, proxyUrl string) *Client {
	ocli := &Client{
		Caller:   caller,
		host:     host,
		proxyUrl: proxyUrl,
		cli:      &http.Client{},
	}
	return ocli
}

func (cli *Client) GetSubject(ctx context.Context, authToken string, subjectId string, body []string) (map[string]interface{}, error) {

	param := map[string]string{
		//"pageIndex": 1,
		//"pageSize":  100,
	}
	if subjectId == "" {
		return nil, errors.New("subject id is required!")
	}
	out, err := cli.GET(ctx, "/v0/subjects/"+subjectId, authToken, 0, param, nil)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (cli Client) SearchMediumSubjectByKeywords(ctx context.Context, authToken string, keywords string, subjectType SubjectType,
	responseGroup string, start int64, maxResults int64) (map[string]interface{}, error) {
	if keywords == "" {
		return nil, errors.New("keywords is required!")
	}
	if subjectType != 0 && !subjectType.IsSupported() {
		return nil, errors.New("subject type is invalid!")
	}
	if start < 0 {
		return nil, errors.New("start is invalid!")
	}
	if maxResults < 0 {
		return nil, errors.New("maxResults is invalid!")
	} else if maxResults > 25 || maxResults == 0 {
		maxResults = 25
	}

	param := map[string]string{
		"type":          subjectType.ToString(),
		"responseGroup": "small",
		"start":         strconv.FormatInt(start, 10),
		"max_results":   strconv.FormatInt(maxResults, 10),
	}
	out, err := cli.POST(ctx, "/search/subject/"+keywords, authToken, 0, param, nil)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (cli *Client) callEncodedUrl(ctx context.Context, method, url string,
	params map[string]string, headers map[string]string, body map[string]string, retryCount uint) (int, string, error) {
	if params == nil {
		params = make(map[string]string)
	}
	return httpcli.HttpFromUrlEncode(ctx, cli.cli, method, url, params, headers, body, retryCount)
}

func (cli *Client) callFromUrlEncode(ctx context.Context, method, absolutePath, authToken string, retry uint, in interface{}, out interface{}) error {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	if authToken != "" {
		headers["Authorization"] = authToken
	}

	if cli.concurrencyChan != nil {
		cli.concurrencyChan <- true
		defer func() {
			if cli.concurrencyChan != nil {
				<-cli.concurrencyChan
			}
		}()
	}
	if len(headers["Content-Type"]) == 0 {
		headers["Content-Type"] = "application/json"
	}
	for {
		_, ok := in.(map[string]string)
		if !ok {
			errMsg := errors.New("in is not map[string]string!")
			return errMsg
		}
		statusCode, content, err := cli.callEncodedUrl(ctx, method, cli.host+absolutePath, //content 为body raw string
			nil, headers, in.(map[string]string), 1)
		if err != nil {
			return err
		}
		// 校验统一抓取返回的结果
		if statusCode != 200 {
			return err
		}
		resp := &respType{}
		// todo [refine] 如何才能优美并整合地解析？
		if err := json.Unmarshal([]byte(content), &resp); err != nil || resp.Data == nil {
			resp = &respType{
				Data: []byte(content),
			}
		}
		if out != nil {
			if err := json.Unmarshal(resp.Data, out); err != nil {
				return err
			}
		}
		return nil
	}
}

func (cli *Client) POST(ctx context.Context, absolutePath, authToken string, retry uint, param map[string]string, in map[string]string) (map[string]interface{}, error) {
	if cli.skip {
		return nil, errors.New("skip is true!")
	}
	if param != nil {
		absolutePath = fmt.Sprintf("%s?", absolutePath)
		for k, v := range param {
			if v == "" {
				continue
			}
			absolutePath = fmt.Sprintf("%s%s=%s&", absolutePath, k, v)
		}
		absolutePath = strings.TrimSuffix(absolutePath, "&")
	}
	out, err1 := cli.callJson(ctx, "POST", absolutePath, authToken, retry, in)
	return out, err1
}

func (cli *Client) GET(ctx context.Context, absolutePath, authToken string, retry uint, param map[string]string, in interface{}) (map[string]interface{}, error) {
	if cli.skip {
		return nil, errors.New("skip is true!")
	}
	if param != nil {
		absolutePath = fmt.Sprintf("%s?", absolutePath)
		for k, v := range param {
			absolutePath = fmt.Sprintf("%s%s=%s&", absolutePath, k, v)
		}
		absolutePath = strings.TrimSuffix(absolutePath, "&")
	}
	out, err1 := cli.callJson(ctx, "GET", absolutePath, authToken, retry, []byte{})
	return out, err1
}

func (cli *Client) callJson(ctx context.Context, method, absolutePath, authToken string, retry uint, in interface{}) (map[string]interface{}, error) {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return nil, err
		}
	}
	//tokenJson := "\n{\n  \"access_token\": \""+authToken+"\"}"
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	if authToken != "" {
		headers["Authorization"] = "Bearer " + authToken /*+ "?scope=read"*/
		//fmt.Println(headers["Authorization"])
	}
	headers["User-Agent"] = "sai/my-private-project"
	out, err1 := cli.Call(ctx, method, absolutePath, nil, headers, retry, body)
	return out, err1
}

func (cli *Client) call(ctx context.Context, method, url string,
	params map[string]string, headers map[string]string, body []byte, retryCount uint) (int, string, error) {
	if params == nil {
		params = make(map[string]string)
	}

	return httpcli.HttpWithCli(ctx, cli.cli, method, url, params, headers, body, retryCount)
}

func (cli *Client) Call(ctx context.Context, method, absolutePath string,
	param map[string]string, headers map[string]string, retry uint, body []byte) (map[string]interface{}, error) {
	if cli.concurrencyChan != nil {
		cli.concurrencyChan <- true
		defer func() {
			if cli.concurrencyChan != nil {
				<-cli.concurrencyChan
			}
		}()
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	if len(headers["Content-Type"]) == 0 {
		headers["Content-Type"] = "application/json"
	}

	//retryN := uint(0)
	for {
		statusCode, content, err := cli.call(ctx, method, cli.host+absolutePath, //content 为body raw string
			param, headers, body, 1)
		if err != nil {
			return nil, err
		}
		if statusCode != http.StatusOK {
			errMsg := errors.New(string(content))
			return nil, errMsg
		}
		var outData map[string]interface{}
		jsonErr2 := json.Unmarshal([]byte(content), &outData)
		if jsonErr2 != nil {
			return nil, jsonErr2
		}
		return outData, nil
	}
}
