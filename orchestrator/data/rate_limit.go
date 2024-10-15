package data

const (
	RateLimitTypeDisabled RateLimitType = "disabled"
	RateLimitTypeDomain   RateLimitType = "per_domain" // domain falls back to IP if no domain is available
	RateLimitTypeIP       RateLimitType = "per_ip"
)

type RateLimitType = string
