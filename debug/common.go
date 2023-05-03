package debug

import (
	"context"
	"fmt"
	"log"
	"reflect"

	gops "github.com/mitchellh/go-ps"
	"github.com/robomotionio/robomotion-go/proto"
	"google.golang.org/grpc"
)

func getRPCAddr() string {
	tabs := GetNetStatPorts(SS_LISTENING, "robomotion-runner")
	tabs = filterTabs(tabs)

	switch len(tabs) {
	case 0:
		return ""
	case 1:
		return tabs[0].LocalAddress
	default:
		return selectTab(tabs)
	}
}

func filterTabs(tabs []*SockTabEntry) []*SockTabEntry {
	var (
		err      error
		filtered = []*SockTabEntry{}
	)

	for _, tab := range tabs {
		addr := tab.LocalAddress
		tab.RobotName, err = getRobotName(addr)
		if err != nil {
			log.Println(err)
			continue
		}

		filtered = append(filtered, tab)
	}

	return filtered
}

func selectTab(tabs []*SockTabEntry) string {
	count := len(tabs)

	robots := ""
	for i, tab := range tabs {
		robots += fmt.Sprintf("%d) %s\n", i+1, tab.RobotName)
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
			return tabs[selected-1].LocalAddress
		}
	}
}

func getRobotName(addr string) (string, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return "", err
	}
	defer conn.Close()

	runnerCli := proto.NewRunnerClient(conn)
	resp, err := runnerCli.RobotName(context.Background(), &proto.Null{})
	if err != nil {
		return "", err
	}

	return resp.RobotName, nil
}

type SocketState string
type List struct {
	arr interface{}
}

type SockTabEntry struct {
	Process      gops.Process
	LocalAddress string
	RobotName    string
}

func NewList(d interface{}) *List {
	arrType := reflect.TypeOf(d)
	kind := arrType.Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		log.Fatalf("Expected an array/slice. Have %+v.", kind.String())
	}

	return &List{arr: d}
}

func (l *List) Filter(predicate func(interface{}) bool) *List {
	arr := reflect.ValueOf(l.arr)
	arrType := arr.Type()
	resultSliceType := reflect.SliceOf(arrType.Elem())
	filtered := reflect.MakeSlice(resultSliceType, 0, 0)

	for i := 0; i < arr.Len(); i++ {
		elem := arr.Index(i)
		if predicate(elem.Interface()) {
			filtered = reflect.Append(filtered, elem)
		}
	}

	return NewList(filtered.Interface())
}

func (l *List) First() interface{} {
	arr := reflect.ValueOf(l.arr)
	if arr.Len() > 0 {
		return arr.Index(0).Interface()
	}
	return nil
}
