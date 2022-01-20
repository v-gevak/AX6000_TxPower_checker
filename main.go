package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/reiver/go-telnet"
)

type Settings struct {
	addr          string
	password      string
	targetTxPower int64
	interfaceName string
}

func main() {
	args := getArgsFromCommandLine()

	conn := connectToRouter(args.addr, args.password)

	defer conn.Close()

	currentTxPower := getCurrentTxPower(conn, args.interfaceName)

	if currentTxPower != args.targetTxPower {
		changeTxPower(conn, args.targetTxPower, args.interfaceName)
		fmt.Printf("TxPower has been changed")
	} else {
		fmt.Printf("TxPower change is not required")
	}

	time.Sleep(time.Second * 3)
}

func getArgsFromCommandLine() Settings {
	addr := flag.String("addr", "192.168.31.1", "server connection address")
	password := flag.String("password", "", "user password")
	targetTxPower := flag.Int64("targetTxPower", 21, "target TX Power")
	interfaceName := flag.String("interface", "wl0", "wireless interface name, e.g wl0 for 5GHz or wl1 for 2.4 GHz")

	flag.Parse()

	return Settings{addr: *addr, password: *password, targetTxPower: *targetTxPower, interfaceName: *interfaceName}
}

func connectToRouter(ip string, password string) *telnet.Conn {
	fullAddr := ip + ":23"
	conn, err := telnet.DialTo(fullAddr)

	checkError("Unable to connect to router by addr"+fullAddr, err)

	ReaderTelnet(conn, []string{"login:"})
	SenderTelnet(conn, "root")
	ReaderTelnet(conn, []string{"Password:"})
	SenderTelnet(conn, password)
	ret := ReaderTelnet(conn, []string{"~#", "incorrect"})

	if strings.Contains(ret, "incorrect") {
		fmt.Println("Wrong password")
		time.Sleep(time.Second * 3)
		os.Exit(0)
	}

	return conn
}

func getCurrentTxPower(conn *telnet.Conn, interfaceName string) int64 {
	SenderTelnet(conn, "iw dev "+interfaceName+" info | grep txpower")

	txpowerRe := regexp.MustCompile(`[0-9]{2}[.][0-9]{2}`)
	txpowerStr := string(txpowerRe.Find([]byte(ReaderTelnet(conn, []string{"~#"}))))
	txPower, err := strconv.ParseFloat(txpowerStr, 32)

	checkError("failed to get current power", err)

	return int64(txPower)
}

func changeTxPower(conn *telnet.Conn, targetTxPower int64, interfaceName string) {
	SenderTelnet(conn, "iwconfig "+interfaceName+" txpower "+strconv.FormatInt(targetTxPower, 10)+"dBm")
	ReaderTelnet(conn, []string{"~#"})
	SenderTelnet(conn, "ifconfig "+interfaceName+" down && ifconfig "+interfaceName+" up")
	ReaderTelnet(conn, []string{"~#"})
}

func ReaderTelnet(conn *telnet.Conn, expect []string) (out string) {
	var buffer [1]byte
	recvData := buffer[:]
	var n int
	var err error

	for {
		n, err = conn.Read(recvData)
		//debug
		//fmt.Println("Bytes: ", n, "Data: ", recvData, string(recvData))
		if n <= 0 || err != nil || containsSubstr(expect, out) {
			break
		} else {
			out += string(recvData)
		}
	}
	return out
}

func SenderTelnet(conn *telnet.Conn, command string) {
	var commandBuffer []byte
	for _, char := range command {
		commandBuffer = append(commandBuffer, byte(char))
	}

	var crlfBuffer [2]byte = [2]byte{'\r', '\n'}
	crlf := crlfBuffer[:]

	conn.Write(commandBuffer)
	conn.Write(crlf)
}

func checkError(desc string, err error) {
	if err != nil {
		fmt.Println(desc)
		time.Sleep(time.Second * 3)
		panic(err)
	}
}

func containsSubstr(s []string, e string) bool {
	for _, a := range s {
		if strings.Contains(e, a) {
			return true
		}
	}
	return false
}
