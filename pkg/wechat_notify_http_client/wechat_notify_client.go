package wechat_notify_http_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coolseven/wechatbot-chatgpt/pkg/logger"
	"github.com/coolseven/wechatbot-chatgpt/pkg/util"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

const (
	GET  = "GET"
	POST = "POST"
)

type WechatNotifyHttpClient struct {
	client            *http.Client
	endpoint          string
	wechatWorkSendKey string
}

func NewWechatNotifyHttpClient(wechatWorkSendKey string) *WechatNotifyHttpClient {
	return &WechatNotifyHttpClient{
		client: &http.Client{
			Timeout: time.Duration(10) * time.Second,
		},
		endpoint:          "https://qyapi.weixin.qq.com",
		wechatWorkSendKey: wechatWorkSendKey,
	}
}

func (c WechatNotifyHttpClient) do(ctx context.Context, method, path string, params map[string]interface{}) (*http.Response, error) {
	targetUrl := c.endpoint + path
	req, err := http.NewRequest(method, targetUrl, nil)

	if err != nil {
		return nil, err
	}
	if ctx != nil {
		req.WithContext(ctx)
	}

	query := req.URL.Query()

	var jsonBodyString []byte
	if method == GET {
		for key, value := range params {
			query.Set(key, util.Interface2String(value))
		}
	} else if method == POST {
		req.Header.Add("Content-Type", "application/json")
		jsonBodyString, _ = json.Marshal(params)
		req.Body = ioutil.NopCloser(strings.NewReader(string(jsonBodyString)))
	}

	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(req)

	reqStr, _ := httputil.DumpRequest(req, true)
	respStr := []byte("<nil>")
	if err != nil {
		respStr = []byte(err.Error()) // 初始值
	}
	if resp != nil {
		respStr, _ = httputil.DumpResponse(resp, true)
	}

	logger.Info(fmt.Sprintf("WechatNotifyHttpClient - 调用企业微信通知接口结束 \r\nrequest:\n%s\r\n%s\r\nresponse:\r\n%s", reqStr, jsonBodyString, respStr))

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c WechatNotifyHttpClient) parseResponse(resp *http.Response, expectedResponseStruct interface{}) error {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("wechat-notify-err, statusCode:%v , body:%s, error: %v",
			resp.StatusCode, string(body), err,
		))
	}

	err = json.Unmarshal(body, expectedResponseStruct)
	if err != nil {
		return err
	}

	return nil
}

// SendNotifyAsPlainText 发送普通消息
func (c WechatNotifyHttpClient) SendNotifyAsPlainText(ctx context.Context, message string) error {
	data := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	//resp, err := c.do(ctx, POST, "/cgi-bin/webhook/send?key=ffc3b820-e941-4166-a581-722446196a90", data)
	path := fmt.Sprintf("/cgi-bin/webhook/send?key=%s", c.wechatWorkSendKey)
	resp, err := c.do(ctx, POST, path, data)
	if err != nil {
		return err
	}

	// curl --request POST \
	//  --url 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=ffc3b820-e941-4166-a581-722446196a90' \
	//  --header 'Content-Type: application/json' \
	//  --data '{
	//  "msgtype": "text",
	//  "text": {
	//    "content": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=ffc3b820-e941-4166-a581-722446196a90"
	//  }
	// }'
	//
	//
	// HTTP/2 200
	// date: Fri, 25 Nov 2022 07:28:41 GMT
	// content-type: application/json; charset=UTF-8
	// content-length: 27
	// server: nginx
	//
	// {
	//	"errcode": 0,
	//	"errmsg": "ok"
	// }
	//
	// 失败时:
	// {
	//	"errcode": 40008,
	//	"errmsg": "invalid message type, hint: [1669361230365202485965774], from ip: 112.95.175.146, more info at https://open.work.weixin.qq.com/devtool/query?e=40008"
	// }
	respModel := struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}{}

	err = c.parseResponse(resp, &respModel)
	if err != nil {
		return err
	}

	if respModel.Errmsg != "ok" {
		return fmt.Errorf("wechat-notify-err, err-msg:%v", respModel.Errmsg)
	}
	return nil
}
