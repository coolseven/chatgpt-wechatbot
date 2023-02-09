package gpt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coolseven/wechatbot-chatgpt/config"
	"github.com/coolseven/wechatbot-chatgpt/pkg/logger"
	gogpt "github.com/sashabaranov/go-gpt3"
	"image/png"
	"io"
	"io/ioutil"
	"os"
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

// Completions see https://platform.openai.com/docs/api-reference/completions/create
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

// CreateImageMedia see https://platform.openai.com/docs/api-reference/images/create
func CreateImageMedia(imageDescription string, imageCount int) ([]io.Reader, error) {
	cfg := config.LoadConfig()

	c := gogpt.NewClient(cfg.ApiKey)
	ctx := context.Background()

	req := gogpt.ImageRequest{
		Prompt:         imageDescription,
		N:              imageCount,
		Size:           gogpt.CreateImageSize1024x1024,
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
		unbased, err := base64.StdEncoding.DecodeString(dataInner.B64JSON)
		if err != nil {
			return localImageFiles, err
		}

		r := bytes.NewReader(unbased)
		im, err := png.Decode(r)
		if err != nil {
			return localImageFiles, err
		}

		dir, _ := os.Getwd() // 代码根目录
		f, _ := ioutil.TempFile(dir, "debugging-"+"*.png")

		err = png.Encode(f, im)
		if err != nil {
			return localImageFiles, err
		}
		err = f.Sync()
		_, err = f.Seek(0, 0)
		localImageFiles = append(localImageFiles, f)
	}

	return localImageFiles, err
}
