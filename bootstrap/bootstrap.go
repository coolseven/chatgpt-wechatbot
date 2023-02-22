package bootstrap

import (
	"context"
	"fmt"
	"github.com/coolseven/wechatbot-chatgpt/config"
	"github.com/coolseven/wechatbot-chatgpt/handlers"
	"github.com/coolseven/wechatbot-chatgpt/pkg/logger"
	"github.com/coolseven/wechatbot-chatgpt/pkg/wechat_notify_http_client"
	"github.com/eatmoreapple/openwechat"
	"io"
	"time"
)

func Run() {
	//bot := openwechat.DefaultBot()
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式，上面登录不上的可以尝试切换这种模式

	// 注册消息处理函数
	handler, err := handlers.NewHandler()
	if err != nil {
		logger.Danger("register error: %v", err)
		return
	}
	bot.MessageHandler = handler

	// 注册登陆二维码回调
	bot.UUIDCallback = handlers.QrCodeCallBack

	// 创建热存储容器对象
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer func(reloadStorage io.ReadWriteCloser) {
		_ = reloadStorage.Close()
	}(reloadStorage)

	// 设置设备id
	bot.SetDeviceId(config.LoadConfig().DeviceId)

	// 登录模式的区别: https://openwechat.readthedocs.io/zh/latest/bot.html

	// 执行热登录
	//err = bot.HotLogin(reloadStorage)
	//if err != nil {
	//	logger.Warning(fmt.Sprintf("login error: %v ", err))
	//	return
	//}

	logger.Info("login in...")

	// 执行免扫码登录
	err = bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption())
	if err != nil {
		logger.Warning(fmt.Sprintf("login error: %v ", err))
		return
	}

	// 定时检测 bot 的在线状态, 当离线时, 通过企业微信进行告警
	startedAt := time.Now()
	go func() {
		notified := false
		wechatWorkClient := wechat_notify_http_client.NewWechatNotifyHttpClient(config.LoadConfig().WechatWorkSendKey)
		for {
			if notified {
				return
			}
			time.Sleep(time.Second * 30)
			lifeSpanInHours := time.Now().Sub(startedAt).Hours()
			if bot.Alive() {
				logger.Info(fmt.Sprintf("service has been alive for %f hours", lifeSpanInHours))
				continue
			} else {
				err := wechatWorkClient.SendNotifyAsPlainText(context.Background(), fmt.Sprintf("coolseven@aliyun, wechat-gpt is dead after %f hours", lifeSpanInHours))
				if err != nil {
					logger.Info(fmt.Sprintf("调用企业微信告警失败: %s", err.Error()))
				}
				notified = true
				return
			}
		}
	}()

	defer func() {
		wechatWorkClient := wechat_notify_http_client.NewWechatNotifyHttpClient(config.LoadConfig().WechatWorkSendKey)
		lifeSpanInHours := time.Now().Sub(startedAt).Hours()
		if panicErr := recover(); panicErr != nil {
			_ = wechatWorkClient.SendNotifyAsPlainText(context.Background(), fmt.Sprintf("coolseven@aliyun, wechat-gpt has panicErr after %f hours", lifeSpanInHours))
			logger.Danger(fmt.Sprintf("service panic: %v", panicErr))
		}

		err := wechatWorkClient.SendNotifyAsPlainText(context.Background(), fmt.Sprintf("coolseven@aliyun, wechat-gpt is dead after %f hours", lifeSpanInHours))
		if err != nil {
			logger.Info(fmt.Sprintf("调用企业微信告警失败: %s , %s", err.Error(), fmt.Sprintf("coolseven@aliyun, wechat-gpt is dead after %f minutes", lifeSpanInHours)))
		}
	}()

	// 服务启动成功通知
	wechatWorkClient := wechat_notify_http_client.NewWechatNotifyHttpClient(config.LoadConfig().WechatWorkSendKey)
	err = wechatWorkClient.SendNotifyAsPlainText(context.Background(), "coolseven@aliyun, wechat-gpt has started!")
	if err != nil {
		logger.Info(fmt.Sprintf("调用企业微信告警失败: %s, %s", err.Error(), "coolseven@aliyun, wechat-gpt has started!"))
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	logger.Info("service started...")
	bot.Block()

	lifeSpanInHours := time.Now().Sub(startedAt).Hours()
	err = wechatWorkClient.SendNotifyAsPlainText(context.Background(), fmt.Sprintf("coolseven@aliyun, wechat-gpt is dead after %f hours", lifeSpanInHours))
	if err != nil {
		logger.Info(fmt.Sprintf("调用企业微信告警失败: %s, %s", err.Error(), fmt.Sprintf("coolseven@aliyun, wechat-gpt is dead after %f hours", lifeSpanInHours)))
	}
}
