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

	switch len(tabs) {
	case 0:
		return ""
	case 1:
		return tabs[0].LocalAddress
	default:
		return selectTab(tabs)
	}
}

func selectTab(tabs []*SockTabEntry) string {
	count := len(tabs)

	robots := ""
	for i, tab := range tabs {
		addr := tab.LocalAddress
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
			return tabs[selected-1].LocalAddress
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

type SocketState string
type List struct {
	arr interface{}
}

type SockTabEntry struct {
	Process      gops.Process
	LocalAddress string
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
