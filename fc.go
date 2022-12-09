package gofc

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

type FCUtil struct {
	// Opts FCOptions
}

// type FCOptions struct {
// 	Timeout   time.Duration // Millisecond
// 	ForceMPIO bool
// }

type TargetPort struct {
	PortName string
	NodeName string
}

type Device struct {
	Name, Size            string
	Type, State           string
	Vendor, Model, Serial string
}

func (fc *FCUtil) GetTargetPorts() []TargetPort {
	return getTargetPorts()
}

func (fc *FCUtil) GetDevices(tnodeName string, lun uint64) (map[string]*Device, error) {
	if !strings.HasPrefix(tnodeName, "0x") {
		tnodeName = "0x" + tnodeName
	}
	portNames := getPortnamesByNodename(tnodeName, nil)
	return getDevicesByPortNamesLun(portNames, lun)
}

func (fc *FCUtil) GetDevicesByDevPath(devPath string) (map[string]*Device, error) {
	return getDevicesByDevPath(devPath)
}

func (fc *FCUtil) RescanHost() {
	fcHostPath := "/sys/class/fc_host/"
	if dirs, err := ioutil.ReadDir(fcHostPath); err == nil {
		for _, f := range dirs {
			statef := fcHostPath + f.Name() + "/port_state"
			data, err := os.ReadFile(statef)
			content := strings.Trim(string(data), "\n")
			glog.V(5).Infof("[RescanDisk] statef(%s) content(%s)", statef, content)
			if err == nil && content == "Online" {
				devFile := fcHostPath + f.Name() + "/issue_lip"
				glog.V(4).Infof("[RescanDisk] echo 1 > %s", devFile)
				if err = writeDeviceFile(devFile, "1"); err != nil {
					glog.Errorf("Failed to echo 1 > %s, err: %v", devFile, err)
				}

				devFile = "/sys/class/scsi_host/" + f.Name() + "/scan"
				glog.V(4).Infof("[RescanDisk] echo \"- - -\" > %s", devFile)
				if err = writeDeviceFile(devFile, "- - -"); err != nil {
					glog.Errorf("Failed to echo \"- - -\" > %s, err: %v", devFile, err)
				}
			}
		}
	}
}

func (fc *FCUtil) RemoveDisk(devPath string) error {
	if strings.HasPrefix(devPath, "/dev/") {
		devices, _ := getDevicesByDevPath(devPath)
		for devName, dev := range devices {
			glog.V(4).Infof("[RemoveDisk] name(%s) dev(%+v)", devName, dev)

			// devName := dev[5:]
			devFile := fmt.Sprintf("/sys/block/%s/device/state", devName)
			if err := writeDeviceFile(devFile, "offline\n"); err != nil {
				return err
			}

			devFile = fmt.Sprintf("/sys/block/%s/device/delete", devName)
			if err := writeDeviceFile(devFile, "1"); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("[RemoveDisk] invalid dev path: %s\n", devPath)
	}

	return nil
}
