package gpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coolseven/wechatbot-chatgpt/config"
	"github.com/coolseven/wechatbot-chatgpt/pkg/logger"
	gogpt "github.com/sashabaranov/go-gpt3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const BASEURL = "https://api.openai.com/v1/"

// ChatGPTResponseBody 请求体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChoiceItem           `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

type ChoiceItem struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	Logprobs     int    `json:"logprobs"`
	FinishReason string `json:"finish_reason"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt"`
	MaxTokens        uint    `json:"max_tokens"`
	Temperature      float64 `json:"temperature"`
	TopP             int     `json:"top_p"`
	FrequencyPenalty int     `json:"frequency_penalty"`
	PresencePenalty  int     `json:"presence_penalty"`
}

// Completions gtp文本模型回复
//curl https://api.openai.com/v1/completions
//-H "Content-Type: application/json"
//-H "Authorization: Bearer your chatGPT key"
//-d '{"model": "text-davinci-003", "prompt": "give me good song", "temperature": 0, "max_tokens": 7}'
func CompletionsDeprecated(msg string) (string, error) {
	cfg := config.LoadConfig()
	requestBody := ChatGPTRequestBody{
		Model:            cfg.Model,
		Prompt:           msg,
		MaxTokens:        cfg.MaxTokens,
		Temperature:      cfg.Temperature,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	requestData, err := json.Marshal(requestBody)

	if err != nil {
		return "", err
	}
	logger.Info(fmt.Sprintf("request gpt json string : %v", string(requestData)))
	req, err := http.NewRequest("POST", BASEURL+"completions", bytes.NewBuffer(requestData))
	if err != nil {
		return "", err
	}

	apiKey := config.LoadConfig().ApiKey
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		body, _ := ioutil.ReadAll(response.Body)
		return "", errors.New(fmt.Sprintf("请求GTP出错了，gpt api status code not equals 200,code is %d ,details:  %v ", response.StatusCode, string(body)))
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	logger.Info(fmt.Sprintf("response gpt json string : %v", string(body)))

	gptResponseBody := &ChatGPTResponseBody{}
	log.Println(string(body))
	err = json.Unmarshal(body, gptResponseBody)
	if err != nil {
		return "", err
	}

	var reply string
	if len(gptResponseBody.Choices) > 0 {
		reply = gptResponseBody.Choices[0].Text
	}
	logger.Info(fmt.Sprintf("gpt response text: %s ", reply))
	return reply, nil
}

func Completions(input string) (string, error) {
	cfg := config.LoadConfig()

	c := gogpt.NewClient(cfg.ApiKey)
	ctx := context.Background()

	req := gogpt.CompletionRequest{
		Model:            gogpt.GPT3TextDavinci003,
		MaxTokens:        int(cfg.MaxTokens),
		Prompt:           input,
		Temperature:      float32(cfg.Temperature),
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	resp, err := c.CreateCompletion(ctx, req)
	if err != nil {
		return "", errors.New(fmt.Sprintf("请求GTP出错了，gpt api err: %v ", err))
	}
	responseBodyString, _ := json.Marshal(resp)
	logger.Info(fmt.Sprintf("response gpt json string : %s", string(responseBodyString)))

	return resp.Choices[0].Text, nil
}

func CreateImage(input string) ([]string, error) {
	cfg := config.LoadConfig()

	c := gogpt.NewClient(cfg.ApiKey)
	ctx := context.Background()

	req := gogpt.ImageRequest{
		Prompt:         input,
		N:              2,
		Size:           gogpt.CreateImageSize512x512,
		ResponseFormat: gogpt.CreateImageResponseFormatURL,
		User:           "",
	}
	resp, err := c.CreateImage(ctx, req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("请求GTP出错了，gpt api err: %v ", err))
	}
	//responseBodyString, _ := json.Marshal(resp)
	//logger.Info(fmt.Sprintf("response gpt json string : %s", string(responseBodyString)))
	var imageUrls []string
	for _, dataInner := range resp.Data {
		//// 将图片base64到本地临时文件中
		//dir, _ := os.Getwd() // 在服务器上是 pod 根目录 (/) , 在本机上是代码根目录
		//f, _ := ioutil.TempFile(dir, "debugging-"+"*.png")
		////localTempFile := f.Name()
		//_, err = f.WriteString(dataInner.B64JSON)
		//fmt.Println(f.Name())
		//fmt.Println("-----------")

		fmt.Println("-----------")
		fmt.Println(dataInner.URL)
		imageUrls = append(imageUrls, dataInner.URL)
	}

	return imageUrls, nil
}

func CreateImageMedia(input string) ([]io.Reader, error) {
	cfg := config.LoadConfig()

	c := gogpt.NewClient(cfg.ApiKey)
	ctx := context.Background()

	req := gogpt.ImageRequest{
		Prompt:         input,
		N:              2,
		Size:           gogpt.CreateImageSize512x512,
		ResponseFormat: gogpt.CreateImageResponseFormatB64JSON,
		User:           "",
	}
	resp, err := c.CreateImage(ctx, req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("请求GTP出错了，gpt api err: %v ", err))
	}

	var localImageFiles []io.Reader
	for _, dataInner := range resp.Data {
		// 将图片base64到本地临时文件中
		dir, _ := os.Getwd() // 在服务器上是 pod 根目录 (/) , 在本机上是代码根目录
		f, _ := ioutil.TempFile(dir, "debugging-"+"*.png")
		//localTempFile := f.Name()
		_, err = f.WriteString("data:image/png;base64," + dataInner.B64JSON)
		fmt.Println(f.Name())
		fmt.Println("-----------")
		err = f.Sync()
		_, err = f.Seek(0, 0)

		localImageFiles = append(localImageFiles, f)
	}

	return localImageFiles, err
}
