package gofc

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

func getTargetPorts() []TargetPort {
	var ports []TargetPort
	fcRemotePath := "/sys/class/fc_remote_ports/"
	if dirs, err := ioutil.ReadDir(fcRemotePath); err == nil {
		for _, dir := range dirs {
			portName, err1 := os.ReadFile(fcRemotePath + dir.Name() + "/port_name")
			nodeName, err2 := os.ReadFile(fcRemotePath + dir.Name() + "/node_name")
			if err1 == nil && err2 == nil {
				ports = append(ports, TargetPort{strings.Trim(string(portName), "\n"), strings.Trim(string(nodeName), "\n")})
			}
		}
	}

	return ports
}

func getNodenameByPortname(portname string, ports []TargetPort) string {
	if ports == nil {
		ports = getTargetPorts()
	}

	for _, port := range ports {
		if port.PortName == portname {
			return port.NodeName
		}
	}

	return ""
}

func getPortnamesByNodename(nodename string, ports []TargetPort) []string {
	if ports == nil {
		ports = getTargetPorts()
	}

	var portnames []string
	for _, port := range ports {
		if port.NodeName == nodename {
			portnames = append(portnames, port.PortName)
		}
	}

	return portnames
}

func getByPathNameFromDevPath(devPath string) (string, error) {
	prefixDir := "/dev/disk/by-path/"
	files, err := ioutil.ReadDir(prefixDir)
	if err != nil {
		return "", fmt.Errorf("Failed to ReadDir: %v", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "pci-") && strings.Contains(file.Name(), "-fc-") {
			fi, _ := os.Lstat(prefixDir + file.Name())
			if fi.Mode()&fs.ModeSymlink != 0 {
				link, _ := os.Readlink(prefixDir + file.Name())
				// fmt.Printf("link: %s, err=%v \n", link, err)

				realDev, _ := filepath.Abs(prefixDir + link)
				// fmt.Printf("real: %s\n", realDev)
				if realDev == devPath {
					glog.V(4).Infof("[getByPathNameFromDevPath] Found %s ==> %s", devPath, file.Name())
					return file.Name(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("No FC device path found. (dev=%s)", devPath)
}

func parseByPathName(s string) (port, lun string, err error) {
	tokens := strings.Split(s, "-")
	// The valid format example, "pci-0000:03:00.2-fc-0x2a00001378d485e0-lun-0"
	// if tokens[0] == "pci" && tokens[2] == "fc" && tokens[len(tokens)-2] == "lun" {
	if tokens[0] == "pci" && tokens[len(tokens)-4] == "fc" && tokens[len(tokens)-2] == "lun" {
		port = tokens[len(tokens)-3]
		lun = tokens[len(tokens)-1]
		return port, lun, nil
	} else {
		return "", "", fmt.Errorf("Invalid device format: %s\n", s)
	}
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
					glog.Errorf("Failed to get device(%s) info, err: %v \n", devicePath, err)
				}
			}
		}
	}

	return devMap, nil
}

func getDevicesByDevPath(devPath string) (map[string]*Device, error) {
	bypath, err := getByPathNameFromDevPath(devPath)
	if err != nil {
		return nil, err
	}
	portname, lun, err := parseByPathName(bypath)
	if err != nil {
		return nil, err
	}

	ports := getTargetPorts()
	nodename := getNodenameByPortname(portname, ports)
	portnames := getPortnamesByNodename(nodename, ports)

	var bypathnames []string
	for _, portname := range portnames {
		bypathname := fmt.Sprintf("-fc-%s-lun-%s", portname, lun)
		bypathnames = append(bypathnames, bypathname)
	}
	glog.V(4).Infof("[getDevicesByDevPath] devPath(%s) ==> bypathnames=%v\n", devPath, bypathnames)

	var devs []*Device
	devMap := make(map[string]*Device)
	prefixDir := "/dev/disk/by-path/"
	files, err := ioutil.ReadDir(prefixDir)
	if err != nil {
		return nil, fmt.Errorf("Failed to ReadDir %s, err: %v\n", prefixDir, err)
	}

	for _, file := range files {
		devicePath := prefixDir + file.Name()
		pos := strings.Index(file.Name(), "-fc-")
		if pos >= 0 && strings.HasPrefix(file.Name(), "pci-") {
			if contains(bypathnames, file.Name()[pos:]) {
				args := []string{"-rn", "-o", "NAME,KNAME,PKNAME,TYPE,STATE,SIZE,VENDOR,MODEL,WWN"}
				out, err := execCmd("lsblk", append(args, []string{devicePath}...)...)
				if err == nil {
					lines := strings.Split(strings.Trim(string(out), "\n"), "\n")
					for _, line := range lines {
						tokens := strings.Split(line, " ")
						glog.V(2).Infof("[getDevicesByDevPath] deviceInfo %+v\n", tokens)
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
					glog.Errorf("[getDevicesByDevPath] Failed to get device(%s) information, err: %v \n", devicePath, err)
				}
			}
		}
	}

	return devMap, nil
}

// Deprecated. lsblk can't get WWN/SERIAL in the container
/*func getVendorModelSerial(devPath string) (string, string, string, error) {
	line, err := lsblkCmd(devPath, "VENDOR,MODEL,WWN")
	if err != nil {
		return "", "", "", err
	}

	tokens := strings.Split(line, " ")
	return tokens[0], tokens[1], tokens[2], nil
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
				glog.Errorf("Failed to get device(%s) info, err: %v \n", devicePath, err)
			}
		}
	}

	return devMap, nil
}*/
