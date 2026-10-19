package capture

import (
	"errors"
	"net/url"
	"strings"
)

var ErrInvalidShareURL = errors.New("仅支持三国杀官网分享链接")

type ShareLink struct {
	ShareURL string `json:"share_url"`
	DeepLink string `json:"deep_link"`
}

func ParseShareURL(raw string) (ShareLink, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "www.sanguosha.cn" || parsed.Path != "/share/index.html" {
		return ShareLink{}, ErrInvalidShareURL
	}

	reqData := parsed.Query().Get("reqdata")
	if reqData == "" {
		return ShareLink{}, errors.New("分享链接缺少 reqdata 参数")
	}
	deepLink, err := url.Parse(reqData)
	if err != nil || !strings.EqualFold(deepLink.Scheme, "yokasgs") {
		return ShareLink{}, errors.New("分享链接中的游戏唤起地址无效")
	}
	if deepLink.Query().Get("payload") == "" {
		return ShareLink{}, errors.New("分享链接中的对局标识为空")
	}

	return ShareLink{ShareURL: parsed.String(), DeepLink: reqData}, nil
}
