package debug

import (
	"log"
	"strings"

	"github.com/cakturk/go-netstat/netstat"
)

func getRPCAddr() string {
	tabs, err := netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.Listen && s.Process != nil && strings.Contains(s.Process.Name, "robomotion-runner")
	})

	if err != nil {
		log.Println(err)
		return ""
	}

	if len(tabs) == 0 {
		return ""
	}

	return tabs[0].LocalAddr.String()
}
