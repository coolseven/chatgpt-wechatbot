package handlers

import (
	"errors"
	"fmt"
	"github.com/coolseven/wechatbot-chatgpt/config"
	"github.com/coolseven/wechatbot-chatgpt/gpt"
	"github.com/coolseven/wechatbot-chatgpt/pkg/logger"
	"github.com/coolseven/wechatbot-chatgpt/service"
	"github.com/eatmoreapple/openwechat"
	"strings"
)

var _ MessageHandlerInterface = (*UserMessageHandler)(nil)

// UserMessageHandler 私聊消息处理
type UserMessageHandler struct {
	// 接收到消息
	msg *openwechat.Message
	// 发送的用户
	sender *openwechat.User
	// 实现的用户业务
	service service.UserServiceInterface
}

func UserMessageContextHandler() func(ctx *openwechat.MessageContext) {
	return func(ctx *openwechat.MessageContext) {
		msg := ctx.Message
		handler, err := NewUserMessageHandler(msg)
		if err != nil {
			logger.Warning(fmt.Sprintf("init user message handler error: %s", err))
		}

		// 处理用户消息
		err = handler.handle()
		if err != nil {
			logger.Warning(fmt.Sprintf("handle user message error: %s", err))
		}
	}
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler(message *openwechat.Message) (MessageHandlerInterface, error) {
	sender, err := message.Sender()
	if err != nil {
		return nil, err
	}
	userService := service.NewUserService(c, sender)
	handler := &UserMessageHandler{
		msg:     message,
		sender:  sender,
		service: userService,
	}

	return handler, nil
}

// handle 处理消息
func (h *UserMessageHandler) handle() error {
	if h.msg.IsText() {
		return h.ReplyText()
	}
	return nil
}

// ReplyText 发送文本消息到群
func (h *UserMessageHandler) ReplyText() error {
	logger.Info(fmt.Sprintf("Received User %v Text Msg : %v", h.sender.NickName, h.msg.Content))
	var (
		reply string
		err   error
	)
	// 1.获取上下文，如果字符串为空不处理
	requestText := h.getRequestText()
	if requestText == "" {
		logger.Info("user message is null")
		return nil
	}
	logger.Info(fmt.Sprintf("h.sender.NickName == %+v", h.sender.NickName))
	// 2.向GPT发起请求，如果回复文本等于空,不回复

	imageModeTriggers := []string{
		"生成图片", "生成一张图片", "生成1张图片",
		"生成两张图片", "生成2张图片",
		"生成三张图片", "生成3张图片",
		"再来一张", "再来1张",
		"再来两张", "再来2张",
		"再来三张", "再来3张",
	}

	imageDescription := ""
	imageWanted := false
	imageCount := 1
	for _, imageModeTrigger := range imageModeTriggers {
		if strings.Contains(h.msg.Content, imageModeTrigger) {
			imageWanted = true
			switch imageModeTrigger {
			case "生成图片", "生成一张图片", "生成1张图片":
				imageCount = 1
				imageDescription = strings.TrimPrefix(h.msg.Content, imageModeTrigger)
				h.service.SetUserSessionContext(imageDescription, "")
			case "生成两张图片", "生成2张图片":
				imageCount = 2
				imageDescription = strings.TrimPrefix(h.msg.Content, imageModeTrigger)
				h.service.SetUserSessionContext(imageDescription, "")
			case "生成三张图片", "生成3张图片":
				imageCount = 3
				imageDescription = strings.TrimPrefix(h.msg.Content, imageModeTrigger)
				h.service.SetUserSessionContext(imageDescription, "")
			case "再来一张", "再来1张":
				previousImageDescription := h.service.GetUserSessionContext()
				if previousImageDescription == "" {
					imageWanted = false
					break
				} else {
					imageCount = 1
					imageDescription = previousImageDescription
				}
			case "再来两张", "再来2张":
				previousImageDescription := h.service.GetUserSessionContext()
				if previousImageDescription == "" {
					imageWanted = false
					break
				} else {
					imageCount = 2
					imageDescription = previousImageDescription
				}
			case "再来三张", "再来3张":
				previousImageDescription := h.service.GetUserSessionContext()
				if previousImageDescription == "" {
					imageWanted = false
					break
				} else {
					imageCount = 3
					imageDescription = previousImageDescription
				}
			}
		}
	}

	if imageWanted {
		imageFiles, err := gpt.CreateImageMedia(imageDescription, imageCount)
		if err != nil {
			// 2.1 将GPT请求失败信息输出给用户，省得整天来问又不知道日志在哪里。
			errMsg := fmt.Sprintf("gpt request error: %v", err)
			_, err = h.msg.ReplyText(errMsg)
			if err != nil {
				return errors.New(fmt.Sprintf("response user error: %v ", err))
			}
			return err
		}
		for _, imageFile := range imageFiles {
			// 2.设置上下文，回复用户
			_, err = h.msg.ReplyImage(imageFile)
			if err != nil {
				_, _ = h.msg.ReplyText("[reply image error]: " + err.Error())
				return errors.New(fmt.Sprintf("response user error: %v ", err))
			}
		}
	} else {
		reply, err = gpt.Completions(h.getRequestText())
		if err != nil {
			// 2.1 将GPT请求失败信息输出给用户，省得整天来问又不知道日志在哪里。
			errMsg := fmt.Sprintf("gpt request error: %v", err)
			_, err = h.msg.ReplyText(errMsg)
			if err != nil {
				return errors.New(fmt.Sprintf("response user error: %v ", err))
			}
			return err
		}

		// 2.设置上下文，回复用户
		h.service.SetUserSessionContext(requestText, reply)
		_, err = h.msg.ReplyText(buildUserReply(reply))
		if err != nil {
			return errors.New(fmt.Sprintf("response user error: %v ", err))
		}
	}

	// 3.返回错误
	return err
}

// getRequestText 获取请求接口的文本，要做一些清晰
func (h *UserMessageHandler) getRequestText() string {
	// 1.去除空格以及换行
	requestText := strings.TrimSpace(h.msg.Content)
	requestText = strings.Trim(h.msg.Content, "\n")

	// 2.获取上下文，拼接在一起，如果字符长度超出4000，截取为4000。（GPT按字符长度算），达芬奇3最大为4068，也许后续为了适应要动态进行判断。
	sessionText := h.service.GetUserSessionContext()
	if sessionText != "" {
		requestText = sessionText + "\n" + requestText
	}
	if len(requestText) >= 4000 {
		requestText = requestText[:4000]
	}

	// 3.检查用户发送文本是否包含结束标点符号
	punctuation := ",.;!?，。！？、…"
	runeRequestText := []rune(requestText)
	lastChar := string(runeRequestText[len(runeRequestText)-1:])
	if strings.Index(punctuation, lastChar) < 0 {
		requestText = requestText + "？" // 判断最后字符是否加了标点，没有的话加上句号，避免openai自动补齐引起混乱。
	}

	// 4.返回请求文本
	return requestText
}

// buildUserReply 构建用户回复
func buildUserReply(reply string) string {
	// 1.去除空格问号以及换行号，如果为空，返回一个默认值提醒用户
	textSplit := strings.Split(reply, "\n\n")
	if len(textSplit) > 1 {
		trimText := textSplit[0]
		reply = strings.Trim(reply, trimText)
	}
	reply = strings.TrimSpace(reply)

	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "请求得不到任何有意义的回复，请具体提出问题。"
	}

	// 2.如果用户有配置前缀，加上前缀
	reply = config.LoadConfig().ReplyPrefix + "\n" + reply
	reply = strings.Trim(reply, "\n")

	// 3.返回拼接好的字符串
	return reply
}
