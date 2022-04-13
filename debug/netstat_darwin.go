package debug

import (
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	gops "github.com/mitchellh/go-ps"
)

const (
	SS_UNKNOWN     SocketState = "UNKNOWN"
	SS_CLOSED      SocketState = "CLOSED"
	SS_LISTENING   SocketState = "LISTEN"
	SS_SYN_SENT    SocketState = "SYN_SENT"
	SS_SYN_RECV    SocketState = "SYN_RECV"
	SS_ESTABLISHED SocketState = "ESTABLISHED"
	SS_FIN_WAIT1   SocketState = "FIN_WAIT1"
	SS_FIN_WAIT2   SocketState = "FIN_WAIT2"
	SS_CLOSE_WAIT  SocketState = "CLOSE_WAIT"
	SS_CLOSING     SocketState = "CLOSING"
	SS_LAST_ACK    SocketState = "LAST_ACK"
	SS_TIME_WAIT   SocketState = "TIME_WAIT"
	SS_DELETE_TCB  SocketState = "DELETE_TCB"
)

func GetNetStatPorts(state SocketState, processName string) []*SockTabEntry {
	tabs := []*SockTabEntry{}

	cmd := exec.Command("netstat", "-a", "-n", "-v", "-p", "tcp")
	out, err := cmd.Output()
	if err != nil {
		log.Println(err)
		return nil
	}

	procs, err := gops.Processes()
	if err != nil {
		log.Println(err)
		return nil
	}

	processList := NewList(procs)
	processList = processList.Filter(func(p interface{}) bool {
		proc := p.(gops.Process)
		a := strings.ToLower(proc.Executable())
		b := strings.ToLower(processName)
		return strings.Contains(a, b) || strings.Contains(b, a)
		// Workaround: proc.Executable() might not contain full name of the process,
		// should check if process names match in either way.
	})

	rr := regexp.MustCompile(`\n`)
	rows := rr.Split(string(out), -1)

	for _, row := range rows {
		if row == "" {
			continue
		}

		tr := regexp.MustCompile(`\s+`)
		tokens := tr.Split(row, -1)
		if len(tokens) <= 10 {
			continue
		}

		pid, err := strconv.Atoi(tokens[8])
		if err != nil {
			continue
		}

		proc := processList.Filter(func(p interface{}) bool {
			proc := p.(gops.Process)
			return proc.Pid() == pid
		}).First()

		if tokens[0] == "tcp4" && tokens[5] == string(state) && proc != nil {
			localAddress := tokens[3]
			tabs = append(tabs, &SockTabEntry{
				Process:      proc.(gops.Process),
				LocalAddress: localAddress,
			})
		}
	}

	return tabs
}
