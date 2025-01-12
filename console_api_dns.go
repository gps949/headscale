package headscale

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
)

type DNSData struct {
	Warning           []string            `json:"warnings"`          // TODO:未实现
	Resolvers         []string            `json:"resolvers"`         //域名服务器列表(覆写本地时)
	Domains           []string            `json:"domains"`           //分离DNS设置的域名
	Routes            map[string][]string `json:"routes"`            //分离DNS设置的映射关系
	FallbackResolvers []string            `json:"fallbackResolvers"` //域名服务器列表(不覆写本地时)
	MagicDNS          bool                `json:"magicDNS"`          //是否启用幻域
	HasNextDNS        bool                `json:"hasNextDNS"`        // TODO:未实现
	MagicDNSDomains   []string            `json:"magicDNSDomains"`   //幻域域列表
}

// 接受/admin/api/dns的Get请求，用于查询DNS
func (h *Headscale) CAPIGetDNS(
	w http.ResponseWriter,
	r *http.Request,
) {
	userName := h.verifyTokenIDandGetUser(w, r)
	if userName == "" {
		h.doAPIResponse(w, "用户信息核对失败", nil)
		return
	}
	user, err := h.GetUser(userName)
	if err != nil {
		h.doAPIResponse(w, "查询用户失败:"+err.Error(), nil)
		return
	}
	userDNSCfg, userBaseDomain := user.GetDNSConfig(h.cfg.IPPrefixes)
	dnsData := DNSData{
		Domains:           make([]string, 0),
		Resolvers:         make([]string, 0),
		FallbackResolvers: make([]string, 0),
		Routes:            make(map[string][]string, 0),
		MagicDNS:          userDNSCfg.Proxied,
	}
	for _, domain := range userDNSCfg.Domains {
		if strings.HasSuffix(domain, "in-addr.arpa") || strings.HasSuffix(domain, "ip6.arpa") {
			continue
		}
		dnsData.Domains = append(dnsData.Domains, domain)
	}
	if len(userDNSCfg.Resolvers) > 0 {
		for _, ns := range userDNSCfg.Resolvers {
			dnsData.Resolvers = append(dnsData.Resolvers, ns.Addr)
		}
	} else if len(userDNSCfg.FallbackResolvers) > 0 {
		for _, ns := range userDNSCfg.FallbackResolvers {
			dnsData.FallbackResolvers = append(dnsData.FallbackResolvers, ns.Addr)
		}
	}
	dnsData.MagicDNSDomains = make([]string, 0)
	dnsData.MagicDNSDomains = append(dnsData.MagicDNSDomains, userBaseDomain)
	if len(userDNSCfg.Routes) > 0 {
		for domain, nsl := range userDNSCfg.Routes {
			if strings.HasSuffix(domain, "in-addr.arpa") || strings.HasSuffix(domain, "ip6.arpa") {
				continue
			}
			dnsData.Routes[domain] = make([]string, 0)
			for _, ns := range nsl {
				dnsData.Routes[domain] = append(dnsData.Routes[domain], ns.Addr)
			}
		}
	}
	h.doAPIResponse(w, "", dnsData)
}

// 请求报文：同DNSData查询报文
// 接受/admin/api/dns的Post请求，用于修改DNS设置
func (h *Headscale) CAPIPostDNS(
	w http.ResponseWriter,
	r *http.Request,
) {
	userName := h.verifyTokenIDandGetUser(w, r)
	if userName == "" {
		h.doAPIResponse(w, "用户信息核对失败", nil)
		return
	}
	err := r.ParseForm()
	if err != nil {
		h.doAPIResponse(w, "用户请求解析失败:"+err.Error(), nil)
		return
	}
	reqData := DNSData{}
	json.NewDecoder(r.Body).Decode(&reqData)
	err = h.UpdateDNSConfig(userName, reqData)
	if err != nil {
		h.doAPIResponse(w, "更新用户DNS设置失败", nil)
		return
	}
	h.CAPIGetDNS(w, r)

}

// 注销Key执行DELETE方法api/keys/:Id
func (h *Headscale) CAPIDelDNS(
	w http.ResponseWriter,
	r *http.Request,
) {
	userName := h.verifyTokenIDandGetUser(w, r)
	targetKeyID := strings.TrimPrefix(r.URL.Path, "/admin/api/keys/")
	allKeys, err := h.ListPreAuthKeys(userName)
	if err != nil {
		h.doAPIResponse(w, "查询用户密钥信息失败", nil)
		return
	}
	toDelKeys := make([]PreAuthKey, 0)
	for _, key := range allKeys {
		if key.Key[:12] == targetKeyID {
			toDelKeys = append(toDelKeys, key)
		}
	}
	if len(toDelKeys) == 0 {
		h.doAPIResponse(w, "该密钥不存在", nil)
		return
	} else if len(toDelKeys) > 1 {
		h.doAPIResponse(w, "存在多个密钥具备相同短形式（ID），请联系工作人员", nil)
		return
	}
	err = h.DestroyPreAuthKey(toDelKeys[0])
	if err != nil {
		h.doAPIResponse(w, "执行密钥删除失败", nil)
		return
	}
	h.doAPIResponse(w, "", targetKeyID)
}
