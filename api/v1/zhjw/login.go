package zhjw

import (
	"log/slog"
	"net/url"
	"strings"

	"github.com/W1ndys/easy-qfnu-api-go/common/response"
	"github.com/go-resty/resty/v2"
	"github.com/gin-gonic/gin"
)

// LoginRequest 登录请求参数
type LoginRequest struct {
	UserAccount  string `form:"userAccount" binding:"required"`
	UserPassword string `form:"userPassword" binding:"required"`
	RANDOMCODE   string `form:"RANDOMCODE" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Cookie string `json:"cookie"`
}

// Login 教务系统登录
// 通过后端代理完成登录，解决 Cookie 管理问题
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "参数错误")
		return
	}

	slog.Info("教务系统登录请求",
		"user_account", req.UserAccount,
		"captcha", req.RANDOMCODE,
	)

	// 创建 Resty 客户端
	client := resty.New()

	// 1. 先访问验证码页面获取 JSESSIONID
	captchaResp, err := client.R().
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		Get("http://zhjw.qfnu.edu.cn/verifycode.servlet")

	if err != nil {
		slog.Error("获取验证码失败", "error", err)
		response.Fail(c, "获取验证码失败，请稍后重试")
		return
	}

	// 提取 Cookie
	cookies := captchaResp.Cookies()
	var jsessionid string
	for _, cookie := range cookies {
		if cookie.Name == "JSESSIONID" {
			jsessionid = cookie.Value
			break
		}
	}

	if jsessionid == "" {
		// 尝试从 Set-Cookie header 中提取
		setCookie := captchaResp.Header().Get("Set-Cookie")
		if strings.Contains(setCookie, "JSESSIONID") {
			// 解析 JSESSIONID=xxx; 格式
			parts := strings.Split(setCookie, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "JSESSIONID=") {
					jsessionid = strings.TrimPrefix(part, "JSESSIONID=")
					break
				}
			}
		}
	}

	if jsessionid == "" {
		slog.Error("无法获取 JSESSIONID")
		response.Fail(c, "系统繁忙，请稍后重试")
		return
	}

	slog.Info("获取到 JSESSIONID", "jsessionid_len", len(jsessionid))

	// 2. 发送登录请求
	formData := url.Values{}
	formData.Set("userAccount", req.UserAccount)
	formData.Set("userPassword", req.UserPassword)
	formData.Set("RANDOMCODE", req.RANDOMCODE)

	loginResp, err := client.R().
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Cookie", "JSESSIONID="+jsessionid).
		SetBody(formData.Encode()).
		Post("http://zhjw.qfnu.edu.cn/jsxsd/xk/LoginToXk")

	if err != nil {
		slog.Error("登录请求失败", "error", err)
		response.Fail(c, "登录请求失败，请稍后重试")
		return
	}

	body := loginResp.String()

	// 检查登录结果
	if strings.Contains(body, "验证码错误") || strings.Contains(body, "验证码不正确") {
		response.Fail(c, "验证码错误")
		return
	}

	if strings.Contains(body, "用户名或密码错误") || strings.Contains(body, "密码错误") {
		response.Fail(c, "用户名或密码错误")
		return
	}

	if strings.Contains(body, "用户名不存在") {
		response.Fail(c, "用户名不存在")
		return
	}

	// 检查是否登录成功
	// 登录成功通常会重定向或者返回特定内容
	if strings.Contains(body, "欢迎") ||
		strings.Contains(body, "top.xsxk") ||
		strings.Contains(body, "main_index") ||
		loginResp.StatusCode() == 302 {

		// 登录成功，返回 Cookie
		cookieStr := "JSESSIONID=" + jsessionid

		slog.Info("登录成功", "user_account", req.UserAccount)
		response.Success(c, LoginResponse{
			Cookie: cookieStr,
		})
		return
	}

	// 其他情况，记录日志并返回错误
	slog.Error("登录失败，未知响应",
		"status_code", loginResp.StatusCode(),
		"response_length", len(body),
		"response_preview", truncateString(body, 500),
	)
	response.Fail(c, "登录失败，请检查输入信息")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
