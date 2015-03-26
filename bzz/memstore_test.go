package bzz

import (
	"testing"

	"github.com/ethereum/go-ethereum/blockpool/test"
	"github.com/ethereum/go-ethereum/common"
)

func testMemStore(l int64, branches int64, t *testing.T) {
	m := newMemStore(nil)
	testStore(m, l, branches, t)
}

func TestMemStore128_10000(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(10000, 128, t)
}

func TestMemStore128_1000(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(1000, 128, t)
}

func TestMemStore128_100(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(100, 128, t)
}

func TestMemStore2_100(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(100, 2, t)
}

func TestMemStoreNotFound(t *testing.T) {
	test.LogInit()
	m := newMemStore(nil)
	_, err := m.Get(&common.Hash{})
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}
