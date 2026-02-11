package zap

import "log"

type redirector struct {
	level string
}

// CollectSysLog redirects system log into zap with specified level
func CollectSysLog(level string) {
	log.SetOutput(&redirector{level: level})
}

func (r *redirector) Write(p []byte) (n int, err error) {
	switch r.level {
	case "debug":
		Debug(string(p))
	case "info":
		Info(string(p))
	case "error":
		Error(string(p))
	case "severe":
		Severe(string(p))
	default:
		Info(string(p))
	}
	return len(p), nil
}
