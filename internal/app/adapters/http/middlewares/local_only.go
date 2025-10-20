package middlewares

import (
	"github.com/gin-gonic/gin"
	"net"
	"strings"
)

func (m *Middlewares) LocalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			c.AbortWithStatus(403)
			return
		}

		ip := net.ParseIP(host)
		if ip == nil {
			c.AbortWithStatus(403)
			return
		}

		if !ip.IsLoopback() &&
			!strings.HasPrefix(ip.String(), "192.168.") &&
			!strings.HasPrefix(ip.String(), "10.") {
			c.AbortWithStatus(403)
			return
		}

		c.Next()
	}
}
