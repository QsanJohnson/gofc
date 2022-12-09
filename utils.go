package gofc

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

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
