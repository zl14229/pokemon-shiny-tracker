package api

import (
	"github.com/gin-gonic/gin"
	"net/url"
)

// Multipart writes must not be usable by an unrelated website against localhost.
// Same-origin Vite proxy sends a loopback Origin. CLI requests have none.
func localOrigin(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if origin := c.GetHeader("Origin"); origin != "" {
		u, e := url.Parse(origin)
		if e != nil || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") || u.User != nil || u.Path != "" || u.RawQuery != "" {
			abort(c, 403, "forbidden", "不接受外部网页访问个人数据操作")
			return
		}
	} else if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		abort(c, 403, "forbidden", "不接受跨站请求")
		return
	}
	c.Next()
}
