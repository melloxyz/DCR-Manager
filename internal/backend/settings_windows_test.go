//go:build windows

package backend

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"os"
	"testing"
	"time"
)

func TestStartupRegistryIsolated(t *testing.T) {
	// Never write to the real Run key.
	path := fmt.Sprintf("Software\\DCRManager-test-%d-%d", os.Getpid(), time.Now().UnixNano())
	k, _, e := registry.CreateKey(registry.CURRENT_USER, path, registry.QUERY_VALUE|registry.SET_VALUE)
	if e != nil {
		t.Fatal(e)
	}
	defer registry.DeleteKey(registry.CURRENT_USER, path)
	defer k.Close()
	const name = "fixture"
	m, r := "\"C:\\test app\\dcr-manager.exe\"", "\"C:\\test app\\dcr-manager.exe\" --start-remote-on-login"
	for _, pair := range [][2]string{{"off", "manager"}, {"manager", "remote"}, {"remote", "off"}, {"off", "manager"}, {"manager", "off"}} {
		if e = setStartup(k, name, m, r, pair[0], pair[1]); e != nil {
			t.Fatal(e)
		}
		if state, e := queryStartup(k, name, m, r); e != nil || state != pair[1] {
			t.Fatal(state, e)
		}
	}
	if e = k.SetStringValue(name, "foreign command"); e != nil {
		t.Fatal(e)
	}
	if e = setStartup(k, name, m, r, "off", "remote"); e == nil {
		t.Fatal("foreign entry overwritten")
	}
	if v, _, e := k.GetStringValue(name); e != nil || v != "foreign command" {
		t.Fatal("foreign entry modified")
	}
	if e = k.SetDWordValue(name, 42); e != nil {
		t.Fatal(e)
	}
	if e = setStartup(k, name, m, r, "off", "manager"); e == nil {
		t.Fatal("non-string entry overwritten")
	}
	if v, _, e := k.GetIntegerValue(name); e != nil || v != 42 {
		t.Fatal("foreign type modified")
	}
}
