package gofc

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"
)

var fc *FCUtil

func TestMain(m *testing.M) {
	fmt.Println("------------Start of TestMain--------------")
	flag.Parse()

	logLevelStr := os.Getenv("GOFC_LOG_LEVEL")
	logLevel, _ := strconv.Atoi(logLevelStr)
	if logLevel > 0 {
		flag.Set("alsologtostderr", "true")
		flag.Set("v", logLevelStr)
	}

	fc = &FCUtil{}

	code := m.Run()
	fmt.Println("------------End of TestMain--------------")
	os.Exit(code)
}

func TestGetTargetPorts(t *testing.T) {
	ports := fc.GetTargetPorts()
	for no, port := range ports {
		fmt.Printf("Port%d: %+v\n", no, port)
	}
}

func TestGetDevicesByTnameLun(t *testing.T) {
	devices, _ := fc.GetDevices("0x2000001378d485e0", 0)
	for no, dev := range devices {
		fmt.Printf("Device(%s): %+v\n", no, dev)
	}
}

func TestGetDevicesByDevPath(t *testing.T) {
	devPath := "/dev/sdb"
	devices, _ := fc.GetDevicesByDevPath(devPath)
	for no, dev := range devices {
		fmt.Printf("Device(%s): %+v\n", no, dev)
	}
}

func TestRescanHost(t *testing.T) {
	fc.RescanHost()
}

func TestRemoveDisk(t *testing.T) {
	devPath := "/dev/sdb"
	if err := fc.RemoveDisk(devPath); err != nil {
		t.Fatalf("TestRemoveDisk failed: %v", err)
	}
}
