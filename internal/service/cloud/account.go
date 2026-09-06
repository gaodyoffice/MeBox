package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Account 云盘账号信息（由 strm 账号实现）。
type Account interface {
	GetID() string
	GetProvider() string
	GetConfig() string // JSON 格式的配置
}

// AccountFinder 账号查找接口（由 strm 仓储实现）。
type AccountFinder interface {
	FindByID(ctx context.Context, id string) (Account, error)
}

// AccountFinderAdapter 适配 strm 仓储到 AccountFinder 接口。
type AccountFinderAdapter struct {
	FindByIDFunc func(ctx context.Context, id string) (Account, error)
}

func (a *AccountFinderAdapter) FindByID(ctx context.Context, id string) (Account, error) {
	return a.FindByIDFunc(ctx, id)
}

// SecretDecryptor 敏感字段解密器接口。
type SecretDecryptor interface {
	Decrypt(v string) string
}

// 敏感配置字段列表（与 strm 保持一致）。
var accountSecretKeys = []string{"cookie", "password", "token", "access_token", "refresh_token", "api_key"}

// AccountConfigProvider 账号配置提供者，封装账号仓储、解密器和 HTTP 客户端。
type AccountConfigProvider struct {
	finder AccountFinder
	crypto SecretDecryptor
	client *http.Client
}

// NewAccountConfigProvider 创建账号配置提供者。
func NewAccountConfigProvider(finder AccountFinder, crypto SecretDecryptor, client *http.Client) *AccountConfigProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &AccountConfigProvider{
		finder: finder,
		crypto: crypto,
		client: client,
	}
}

// GetProvider 根据账号 ID 获取云盘 Provider。
func (p *AccountConfigProvider) GetProvider(ctx context.Context, accountID string) (Provider, error) {
	acct, err := p.finder.FindByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("find account %s: %w", accountID, err)
	}
	if acct == nil {
		return nil, fmt.Errorf("account %s not found", accountID)
	}
	return p.ProviderFor(ctx, acct)
}

// ProviderFor 根据账号构建云盘 Provider。
func (p *AccountConfigProvider) ProviderFor(ctx context.Context, acct Account) (Provider, error) {
	cfg, err := p.accountConfig(acct)
	if err != nil {
		return nil, err
	}
	anyCfg := make(map[string]any, len(cfg)+1)
	for k, v := range cfg {
		anyCfg[k] = v
	}
	anyCfg["ua"] = defaultUA
	return New(acct.GetProvider(), anyCfg, p.client)
}

// accountConfig 解析账号配置并解密敏感字段。
func (p *AccountConfigProvider) accountConfig(acct Account) (map[string]string, error) {
	cfg := map[string]string{}
	raw := acct.GetConfig()
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("decode account config: %w", err)
	}
	if p.crypto != nil {
		for _, k := range accountSecretKeys {
			if v, ok := cfg[k]; ok {
				cfg[k] = p.crypto.Decrypt(v)
			}
		}
	}
	return cfg, nil
}

// GetProviderType 根据账号 ID 获取 Provider 类型。
func (p *AccountConfigProvider) GetProviderType(ctx context.Context, accountID string) (string, error) {
	acct, err := p.finder.FindByID(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("find account %s: %w", accountID, err)
	}
	if acct == nil {
		return "", fmt.Errorf("account %s not found", accountID)
	}
	return acct.GetProvider(), nil
}
