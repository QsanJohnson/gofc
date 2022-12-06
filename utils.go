package gofc

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

func writeDeviceFile(devFile, content string) error {
	data := []byte(content)
	return os.WriteFile(devFile, data, 0644)
}

func execCmd(name string, args ...string) (string, error) {
	glog.V(5).Infof("[execCmd] %s, args=%+v \n", name, args)
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	glog.V(5).Infof("[execCmd] Output ==>\n%+v\n", string(out))
	if err != nil {
		return "", fmt.Errorf("%s (%s)\n", strings.TrimRight(string(out), "\n"), err)
	}

	return string(out), err
}

func lsblkCmd(devicePath string, cols string) (string, error) {
	args := []string{"-rn", "-o", cols}
	out, err := execCmd("lsblk", append(args, []string{devicePath}...)...)
	if err != nil {
		return "", err
	}

	val := strings.Trim(string(out), "\n")
	return val, nil
}

func getVendorModelSerial(devPath string) (string, string, string, error) {
	line, err := lsblkCmd(devPath, "VENDOR,MODEL,WWN")
	if err != nil {
		return "", "", "", err
	}

	tokens := strings.Split(line, " ")
	return tokens[0], tokens[1], tokens[2], nil
}

func getDevicesByPortNamesLun(portNames []string, lun uint64) (map[string]*Device, error) {
	var devs []*Device
	devMap := make(map[string]*Device)
	prefixDir := "/dev/disk/by-path/"

	for _, portName := range portNames {
		suffixDevicePath := strings.Join([]string{"fc", portName, "lun", fmt.Sprint(lun)}, "-")
		glog.V(4).Infof("[getDevicesByPortNamesLun] suffixDevicePath=%s\n", suffixDevicePath)

		files, err := ioutil.ReadDir(prefixDir)
		if err != nil {
			return nil, fmt.Errorf("Failed to ReadDir %s, err: %v\n", prefixDir, err)
		}

		for _, file := range files {
			devicePath := prefixDir + file.Name()
			if strings.HasPrefix(file.Name(), "pci-") && strings.HasSuffix(file.Name(), suffixDevicePath) {
				args := []string{"-rn", "-o", "NAME,KNAME,PKNAME,TYPE,STATE,SIZE,VENDOR,MODEL,WWN"}
				out, err := execCmd("lsblk", append(args, []string{devicePath}...)...)
				if err == nil {
					lines := strings.Split(strings.Trim(string(out), "\n"), "\n")
					for _, line := range lines {
						tokens := strings.Split(line, " ")
						glog.V(2).Infof("[getDevicesByPortNamesLun] deviceInfo %+v\n", tokens)
						dev := &Device{
							Name:   tokens[0],
							Type:   tokens[3],
							State:  tokens[4],
							Size:   tokens[5],
							Vendor: tokens[6],
							Model:  tokens[7],
							Serial: tokens[8],
						}
						devs = append(devs, dev)
						devMap[tokens[1]] = dev
					}
				} else {
					fmt.Printf("Failed to get disk path : %v \n", err)
				}
			}
		}
	}

	return devMap, nil
}

func getDevicesByDevPath(devPath string) (map[string]*Device, error) {
	vendor, model, serial, err := getVendorModelSerial(devPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to getVendorModelSerial %s, err: %v\n", devPath, err)
	}

	var devs []*Device
	devMap := make(map[string]*Device)
	prefixDir := "/dev/disk/by-path/"
	files, err := ioutil.ReadDir(prefixDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to ReadDir %s, err: %v\n", prefixDir, err)
	}

	for _, file := range files {
		devicePath := prefixDir + file.Name()
		if strings.HasPrefix(file.Name(), "pci-") && strings.Contains(file.Name(), "-fc-") {
			args := []string{"-rn", "-o", "NAME,KNAME,PKNAME,TYPE,STATE,SIZE,VENDOR,MODEL,WWN"}
			out, err := execCmd("lsblk", append(args, []string{devicePath}...)...)
			if err == nil {
				lines := strings.Split(strings.Trim(string(out), "\n"), "\n")
				for _, line := range lines {
					tokens := strings.Split(line, " ")
					glog.V(2).Infof("[getDevicesByDevPath] deviceInfo %+v\n", tokens)
					if vendor == tokens[6] && model == tokens[7] && serial == tokens[8] {
						dev := &Device{
							Name:   tokens[0],
							Type:   tokens[3],
							State:  tokens[4],
							Size:   tokens[5],
							Vendor: tokens[6],
							Model:  tokens[7],
							Serial: tokens[8],
						}
						devs = append(devs, dev)
						devMap[tokens[1]] = dev
					}
				}
			} else {
				fmt.Printf("Failed to get disk path : %v \n", err)
			}
		}
	}

	return devMap, nil
}
