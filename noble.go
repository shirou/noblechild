package noblechild

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type NobleModule struct {
	Directory string
	HCIPath   string
	L2CAPPath string
}

// build/Release/hci-ble
// build/Release/l2cap-ble

func init() {
	log.SetLevel(log.DebugLevel)
}

func FindNobleModule() (NobleModule, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return NobleModule{}, err
	}

	paths := []string{
		os.Getenv("NOBLE_TOPDIR"),
		dir,
		os.Getenv("HOME"),
	}

	var m NobleModule
	for _, p := range paths {
		dir := path.Join(p, "node_modules", "noble", "build", "Release")

		hci := path.Join(dir, "hci-ble")
		_, err := os.Stat(hci)
		if err != nil {
			continue
		}
		l2cap := path.Join(dir, "l2cap-ble")
		_, err = os.Stat(l2cap)
		if err != nil {
			continue
		}

		m := NobleModule{
			Directory: dir,
			HCIPath:   hci,
			L2CAPPath: l2cap,
		}
		return m, nil
	}

	return m, fmt.Errorf("hci-ble and l2cap-ble not found")
}

// AddrToCommaAddr converts "aaaaaa" to "aa:aa:aa"
func AddrToCommaAddr(orig string) string {
	if strings.Contains(orig, ":") {
		return orig
	}
	var res string
	ret := make([]string, 0, len(orig)/2)
	for i, r := range orig {
		res = res + string(r)
		if i > 0 && (i+1)%2 == 0 {
			ret = append(ret, res)
			res = ""
		}
	}
	return strings.Join(ret, ":")
}
