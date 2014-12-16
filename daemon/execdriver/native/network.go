package native

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/execdriver"
	"github.com/docker/docker/pkg/reexec"
	"github.com/docker/libcontainer/netlink"
	"github.com/docker/libcontainer/system"
)

const (
	nsBin         = "setupnetns"
	nsLoopbackBin = "loopbackupinns"
)

func setIfaceIP(iface *net.Interface, address string) error {
	ip, ipNet, err := net.ParseCIDR(address)
	if err != nil {
		return err
	}
	return netlink.NetworkLinkAddIp(iface, ip, ipNet)
}

type setupError struct {
	Message string
}

func (s setupError) Error() string {
	return s.Message
}

func LoopbackUp(nsPath string) error {
	parent, child, err := newInitPipe()
	if err != nil {
		return err
	}
	defer parent.Close()
	cmd := &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{nsLoopbackBin, nsPath},
	}
	cmd.ExtraFiles = []*os.File{child}
	if err := cmd.Start(); err != nil {
		child.Close()
		return err
	}
	child.Close()
	if err := syscall.Shutdown(int(parent.Fd()), syscall.SHUT_WR); err != nil {
		return err
	}
	var serr *setupError
	if err := json.NewDecoder(parent).Decode(&serr); err != nil && err != io.EOF {
		return err
	}
	if serr != nil {
		return serr
	}
	return nil
}

func init() { reexec.Register(nsLoopbackBin, loopbackUp) }
func loopbackUp() {
	runtime.LockOSThread()
	f := os.NewFile(3, "child")
	var err error
	defer func() {
		if err != nil {
			ioutil.ReadAll(f)
			if err := json.NewEncoder(f).Encode(setupError{Message: err.Error()}); err != nil {
				panic(err)
			}
		}
		f.Close()
	}()
	err = loopbackInNS(os.Args[1])
	return
}

func loopbackInNS(nsPath string) error {
	f, err := os.OpenFile(nsPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed get network namespace fd: %v", err)
	}
	if err := system.Setns(f.Fd(), syscall.CLONE_NEWNET); err != nil {
		return err
	}
	f.Close()
	// up loopback
	lo, err := net.InterfaceByName("lo")
	if err != nil {
		return err
	}
	return netlink.NetworkLinkUp(lo)
}

func SetupChild(nsPath string, n *execdriver.NetworkSettings) error {
	log.Printf("NsPath: %s, Settings: %#v", nsPath, n)
	parent, child, err := newInitPipe()
	if err != nil {
		return err
	}
	defer parent.Close()
	cmd := &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{nsBin, nsPath},
	}
	cmd.ExtraFiles = []*os.File{child}
	if err := cmd.Start(); err != nil {
		child.Close()
		return err
	}
	child.Close()
	if err := json.NewEncoder(parent).Encode(n); err != nil {
		return err
	}
	if err := syscall.Shutdown(int(parent.Fd()), syscall.SHUT_WR); err != nil {
		return err
	}
	var serr *setupError
	if err := json.NewDecoder(parent).Decode(&serr); err != nil && err != io.EOF {
		return err
	}
	if serr != nil {
		return serr
	}
	return nil
}

func newInitPipe() (parent *os.File, child *os.File, err error) {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), "parent"), os.NewFile(uintptr(fds[0]), "child"), nil
}

func init() { reexec.Register(nsBin, setupInterface) }
func setupInterface() {
	runtime.LockOSThread()
	f := os.NewFile(3, "child")
	var err error
	n := &execdriver.NetworkSettings{}
	defer func() {
		if err != nil {
			ioutil.ReadAll(f)
			if err := json.NewEncoder(f).Encode(setupError{Message: err.Error()}); err != nil {
				panic(err)
			}
		}
		f.Close()
	}()
	if err = json.NewDecoder(f).Decode(n); err != nil {
		return
	}
	err = setupInNS(os.Args[1], n)
	return
}

func setupInNS(nsPath string, n *execdriver.NetworkSettings) error {
	f, err := os.OpenFile(nsPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed get network namespace fd: %v", err)
	}
	nsFD := f.Fd()
	name := n.Name
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	if err := netlink.NetworkSetNsFd(iface, int(nsFD)); err != nil {
		return err
	}
	// switching to netns
	err = system.Setns(nsFD, syscall.CLONE_NEWNET)
	f.Close()
	if err != nil {
		return err
	}
	if n.MacAddress != "" {
		if err := netlink.NetworkSetMacAddress(iface, n.MacAddress); err != nil {
			return fmt.Errorf("set %s mac %s", name, err)
		}
	}
	if err := setIfaceIP(iface, n.Address); err != nil {
		return fmt.Errorf("set %s ip %s", name, err)
	}
	if n.IPv6Address != "" {
		if err := setIfaceIP(iface, n.IPv6Address); err != nil {
			return fmt.Errorf("set %s ipv6 %s", name, err)
		}
	}
	if err := netlink.NetworkSetMTU(iface, n.Mtu); err != nil {
		return err
	}
	if err := netlink.NetworkLinkUp(iface); err != nil {
		return err
	}

	if n.Gateway != "" {
		if err := netlink.AddDefaultGw(n.Gateway, name); err != nil {
			return fmt.Errorf("set gateway to %s on device %s failed with %s", n.Gateway, name, err)
		}
	}
	if n.IPv6Gateway != "" {
		if err := netlink.AddDefaultGw(n.IPv6Gateway, name); err != nil {
			return fmt.Errorf("set gateway for ipv6 to %s on device %s failed with %s", n.IPv6Gateway, name, err)
		}
	}
	return nil
}
