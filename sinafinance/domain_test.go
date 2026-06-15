package sinafinance

import (
	"testing"
)

// These tests are offline: they exercise domain metadata and kit wiring.
// Network behaviour is covered in sinafinance_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "sinafinance" {
		t.Errorf("Scheme = %q, want sinafinance", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "sinafinance" {
		t.Errorf("Identity.Binary = %q, want sinafinance", info.Identity.Binary)
	}
}
