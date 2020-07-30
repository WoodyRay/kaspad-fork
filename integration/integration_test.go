package integration

import (
	"testing"
	"time"

	"github.com/kaspanet/kaspad/wire"

	clientpkg "github.com/kaspanet/kaspad/rpc/client"
)

func TestIntegrationBasicSync(t *testing.T) {
	appHarness1, appHarness2, appHarness3, teardown := setup(t)
	defer teardown()

	// Connect nodes in chain: 1 <--> 2 <--> 3
	// So that node 3 doesn't directly get blocks from node 1
	connect(t, appHarness1, appHarness2)
	connect(t, appHarness2, appHarness3)

	blockTemplate, err := appHarness1.rpcClient.GetBlockTemplate(testAddress1, "")
	if err != nil {
		t.Fatalf("Error getting block template: %+v", err)
	}

	block, err := clientpkg.ConvertGetBlockTemplateResultToBlock(blockTemplate)
	if err != nil {
		t.Fatalf("Error parsing blockTemplate: %s", err)
	}

	solveBlock(t, block)

	err = appHarness2.rpcClient.NotifyBlocks()
	if err != nil {
		t.Fatalf("Error from NotifyBlocks: %+v", err)
	}
	app2OnBlockAddedChan := make(chan *wire.BlockHeader)
	appHarness2.rpcClient.onBlockAdded = func(header *wire.BlockHeader) {
		app2OnBlockAddedChan <- header
	}

	err = appHarness3.rpcClient.NotifyBlocks()
	if err != nil {
		t.Fatalf("Error from NotifyBlocks: %+v", err)
	}
	app3OnBlockAddedChan := make(chan *wire.BlockHeader)
	appHarness3.rpcClient.onBlockAdded = func(header *wire.BlockHeader) {
		app3OnBlockAddedChan <- header
	}

	err = appHarness1.rpcClient.SubmitBlock(block, nil)
	if err != nil {
		t.Fatalf("Error submitting block: %s", err)
	}

	var header *wire.BlockHeader
	select {
	case header = <-app2OnBlockAddedChan:
	case <-time.After(defaultTimeout):
		t.Fatalf("Timeout waiting for block added notification on node directly connected to miner")
	}

	if !header.BlockHash().IsEqual(block.Hash()) {
		t.Errorf("Expected block with hash '%s', but got '%s'", block.Hash(), header.BlockHash())
	}

	select {
	case header = <-app3OnBlockAddedChan:
	case <-time.After(defaultTimeout):
		t.Fatalf("Timeout waiting for block added notification on node indirectly connected to miner")
	}

	if !header.BlockHash().IsEqual(block.Hash()) {
		t.Errorf("Expected block with hash '%s', but got '%s'", block.Hash(), header.BlockHash())
	}
}