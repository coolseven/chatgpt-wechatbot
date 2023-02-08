package bootstrap

import (
	"fmt"
	"github.com/coolseven/wechatbot-chatgpt/handlers"
	"github.com/coolseven/wechatbot-chatgpt/pkg/logger"
	"github.com/eatmoreapple/openwechat"
	"io"
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

	// 登录模式的区别: https://openwechat.readthedocs.io/zh/latest/bot.html

	// 执行热登录
	//err = bot.HotLogin(reloadStorage)
	//if err != nil {
	//	logger.Warning(fmt.Sprintf("login error: %v ", err))
	//	return
	//}

	// 执行免扫码登录
	err = bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption())
	if err != nil {
		logger.Warning(fmt.Sprintf("login error: %v ", err))
		return
	}

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}
