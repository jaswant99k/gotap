// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"net"
	"strings"
)

// IPWhitelistConfig holds IP whitelist configuration
type IPWhitelistConfig struct {
	// AllowedIPs is a list of allowed IP addresses or CIDR ranges
	AllowedIPs []string

	// TrustedProxies is a list of trusted proxy IPs
	// When set, the middleware will look at X-Forwarded-For or X-Real-IP headers
	TrustedProxies []string

	// ErrorHandler is called when IP is not whitelisted
	ErrorHandler func(*Context)

	// UseXForwardedFor determines if X-Forwarded-For header should be checked
	// Only works if request comes from a trusted proxy
	UseXForwardedFor bool

	// UseXRealIP determines if X-Real-IP header should be checked
	// Only works if request comes from a trusted proxy
	UseXRealIP bool
}

// IPWhitelist returns an IP whitelist middleware
func IPWhitelist(allowedIPs ...string) HandlerFunc {
	return IPWhitelistWithConfig(IPWhitelistConfig{
		AllowedIPs: allowedIPs,
	})
}

// IPWhitelistWithConfig returns an IP whitelist middleware with config
func IPWhitelistWithConfig(config IPWhitelistConfig) HandlerFunc {
	if len(config.AllowedIPs) == 0 {
		panic("IP whitelist cannot be empty")
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = func(c *Context) {
			c.JSON(403, H{
				"error":   "Forbidden",
				"message": "Your IP address is not allowed to access this resource",
			})
			c.Abort()
		}
	}

	// Parse allowed IPs and CIDR ranges
	allowedNets := make([]*net.IPNet, 0, len(config.AllowedIPs))
	allowedIPsMap := make(map[string]bool)

	for _, ipStr := range config.AllowedIPs {
		ipStr = strings.TrimSpace(ipStr)

		// Check if it's a CIDR range
		if strings.Contains(ipStr, "/") {
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				panic("invalid CIDR range: " + ipStr)
			}
			allowedNets = append(allowedNets, ipNet)
		} else {
			// Single IP
			ip := net.ParseIP(ipStr)
			if ip == nil {
				panic("invalid IP address: " + ipStr)
			}
			allowedIPsMap[ip.String()] = true
		}
	}

	// Parse trusted proxies
	trustedProxiesMap := make(map[string]bool)
	if len(config.TrustedProxies) > 0 {
		for _, proxyIP := range config.TrustedProxies {
			ip := net.ParseIP(strings.TrimSpace(proxyIP))
			if ip == nil {
				panic("invalid trusted proxy IP: " + proxyIP)
			}
			trustedProxiesMap[ip.String()] = true
		}
	}

	return func(c *Context) {
		// Get client IP
		clientIP := getClientIPForWhitelist(c, trustedProxiesMap, config.UseXForwardedFor, config.UseXRealIP)

		// Parse IP
		ip := net.ParseIP(clientIP)
		if ip == nil {
			config.ErrorHandler(c)
			return
		}

		// Check if IP is in allowed list
		if allowedIPsMap[ip.String()] {
			c.Next()
			return
		}

		// Check if IP is in allowed CIDR ranges
		for _, ipNet := range allowedNets {
			if ipNet.Contains(ip) {
				c.Next()
				return
			}
		}

		// IP not whitelisted
		config.ErrorHandler(c)
	}
}

// getClientIPForWhitelist gets the client IP with proxy support
func getClientIPForWhitelist(c *Context, trustedProxies map[string]bool, useXForwardedFor, useXRealIP bool) string {
	// Get remote address
	remoteIP, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		remoteIP = c.Request.RemoteAddr
	}

	// If no trusted proxies configured, return remote IP
	if len(trustedProxies) == 0 {
		return remoteIP
	}

	// Check if request is from a trusted proxy
	if !trustedProxies[remoteIP] {
		return remoteIP
	}

	// Try X-Forwarded-For
	if useXForwardedFor {
		xff := c.Request.Header.Get("X-Forwarded-For")
		if xff != "" {
			// X-Forwarded-For can contain multiple IPs, get the first one
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if clientIP != "" {
					return clientIP
				}
			}
		}
	}

	// Try X-Real-IP
	if useXRealIP {
		xRealIP := c.Request.Header.Get("X-Real-IP")
		if xRealIP != "" {
			return strings.TrimSpace(xRealIP)
		}
	}

	// Fallback to remote IP
	return remoteIP
}

// IPBlacklist returns an IP blacklist middleware
func IPBlacklist(blockedIPs ...string) HandlerFunc {
	return IPBlacklistWithConfig(IPBlacklistConfig{
		BlockedIPs: blockedIPs,
	})
}

// IPBlacklistConfig holds IP blacklist configuration
type IPBlacklistConfig struct {
	// BlockedIPs is a list of blocked IP addresses or CIDR ranges
	BlockedIPs []string

	// TrustedProxies is a list of trusted proxy IPs
	TrustedProxies []string

	// ErrorHandler is called when IP is blacklisted
	ErrorHandler func(*Context)

	// UseXForwardedFor determines if X-Forwarded-For header should be checked
	UseXForwardedFor bool

	// UseXRealIP determines if X-Real-IP header should be checked
	UseXRealIP bool
}

// IPBlacklistWithConfig returns an IP blacklist middleware with config
func IPBlacklistWithConfig(config IPBlacklistConfig) HandlerFunc {
	if len(config.BlockedIPs) == 0 {
		panic("IP blacklist cannot be empty")
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = func(c *Context) {
			c.JSON(403, H{
				"error":   "Forbidden",
				"message": "Your IP address has been blocked",
			})
			c.Abort()
		}
	}

	// Parse blocked IPs and CIDR ranges
	blockedNets := make([]*net.IPNet, 0, len(config.BlockedIPs))
	blockedIPsMap := make(map[string]bool)

	for _, ipStr := range config.BlockedIPs {
		ipStr = strings.TrimSpace(ipStr)

		// Check if it's a CIDR range
		if strings.Contains(ipStr, "/") {
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				panic("invalid CIDR range: " + ipStr)
			}
			blockedNets = append(blockedNets, ipNet)
		} else {
			// Single IP
			ip := net.ParseIP(ipStr)
			if ip == nil {
				panic("invalid IP address: " + ipStr)
			}
			blockedIPsMap[ip.String()] = true
		}
	}

	// Parse trusted proxies
	trustedProxiesMap := make(map[string]bool)
	if len(config.TrustedProxies) > 0 {
		for _, proxyIP := range config.TrustedProxies {
			ip := net.ParseIP(strings.TrimSpace(proxyIP))
			if ip == nil {
				panic("invalid trusted proxy IP: " + proxyIP)
			}
			trustedProxiesMap[ip.String()] = true
		}
	}

	return func(c *Context) {
		// Get client IP
		clientIP := getClientIPForWhitelist(c, trustedProxiesMap, config.UseXForwardedFor, config.UseXRealIP)

		// Parse IP
		ip := net.ParseIP(clientIP)
		if ip == nil {
			c.Next()
			return
		}

		// Check if IP is in blocked list
		if blockedIPsMap[ip.String()] {
			config.ErrorHandler(c)
			return
		}

		// Check if IP is in blocked CIDR ranges
		for _, ipNet := range blockedNets {
			if ipNet.Contains(ip) {
				config.ErrorHandler(c)
				return
			}
		}

		// IP not blacklisted
		c.Next()
	}
}

// CombinedIPFilter combines whitelist and blacklist
// Blacklist is checked first, then whitelist
func CombinedIPFilter(whitelist, blacklist []string) HandlerFunc {
	blacklistMW := IPBlacklist(blacklist...)
	whitelistMW := IPWhitelist(whitelist...)

	return func(c *Context) {
		// Check blacklist first
		blacklistMW(c)
		if c.IsAborted() {
			return
		}

		// Then check whitelist
		whitelistMW(c)
	}
}
