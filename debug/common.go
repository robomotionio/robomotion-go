package debug

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cakturk/go-netstat/netstat"
	"github.com/robomotionio/robomotion-go/proto"
	"google.golang.org/grpc"
)

func getRPCAddr() string {
	tabs, err := netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		return s.State == netstat.Listen && s.Process != nil && strings.Contains(s.Process.Name, "robomotion-runner")
	})

	if err != nil {
		log.Println(err)
		return ""
	}

	switch len(tabs) {
	case 0:
		return ""
	case 1:
		return tabs[0].LocalAddr.String()
	default:
		return selectTab(tabs)
	}
}

func selectTab(tabs []netstat.SockTabEntry) string {
	count := len(tabs)

	robots := ""
	for i, tab := range tabs {
		addr := tab.LocalAddr.String()
		name := getRobotName(addr)
		robots += fmt.Sprintf("%d) %s\n", i+1, name)
	}

	flags := log.Flags()
	defer log.SetFlags(flags)

	log.SetFlags(0)
	log.Printf("\nFound %d robots running on the machine:\n", count)
	log.Printf("%s", robots)

	selected := 0
	log.Printf("Please select a robot to attach (1-%d): ", count)
	for {
		fmt.Scanf("%d", &selected)
		if selected > 0 && selected <= count {
			return tabs[selected-1].LocalAddr.String()
		}
	}
}

func getRobotName(addr string) string {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	runnerCli := proto.NewRunnerClient(conn)
	resp, err := runnerCli.RobotName(context.Background(), &proto.Null{})
	if err != nil {
		log.Println(err)
		return ""
	}

	return resp.RobotName
}
