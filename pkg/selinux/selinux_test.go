package selinux_test

import (
	"github.com/dotcloud/docker/pkg/selinux"
	"os"
	"testing"
)

func testSetfilecon(t *testing.T) {
	if selinux.SelinuxEnabled() {
		tmp := "selinux_test"
		out, _ := os.OpenFile(tmp, os.O_WRONLY, 0)
		out.Close()
		err := selinux.Setfilecon(tmp, "system_u:object_r:bin_t:s0")
		if err == nil {
			t.Log(selinux.Getfilecon(tmp))
		} else {
			t.Log("Setfilecon failed")
			t.Fatal(err)
		}
		os.Remove(tmp)
	}
}

func TestSELinux(t *testing.T) {
	var (
		err            error
		plabel, flabel string
	)

	if selinux.SelinuxEnabled() {
		t.Log("Enabled")
		plabel, flabel = selinux.GetLxcContexts()
		t.Log(plabel)
		t.Log(flabel)
		plabel, flabel = selinux.GetLxcContexts()
		t.Log(plabel)
		t.Log(flabel)
		t.Log("getenforce ", selinux.SelinuxGetEnforce())
		t.Log("getenforcemode ", selinux.SelinuxGetEnforceMode())
		pid := os.Getpid()
		t.Log("PID:%d MCS:%s\n", pid, selinux.IntToMcs(pid, 1023))
		t.Log(selinux.Getcon())
		t.Log(selinux.Getfilecon("/etc/passwd"))
		err = selinux.Setfscreatecon("unconfined_u:unconfined_r:unconfined_t:s0")
		if err == nil {
			t.Log(selinux.Getfscreatecon())
		} else {
			t.Log("setfscreatecon failed", err)
			t.Fatal(err)
		}
		err = selinux.Setfscreatecon("")
		if err == nil {
			t.Log(selinux.Getfscreatecon())
		} else {
			t.Log("setfscreatecon failed", err)
			t.Fatal(err)
		}
		t.Log(selinux.Getpidcon(1))
		t.Log(selinux.GetSelinuxMountPoint())
	} else {
		t.Log("Disabled")
	}
}
