package etherstream_test

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	ether "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/pascaldekloe/etherstream"
)

func ExampleReader() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ethereumRPCURL := "wss://eth-goerli.g.alchemy.com/v2/8DXx4tv3-IXWBgwrtyhhmElxV0_VM9YK"
	ethereumRPCClient, err := ethclient.DialContext(ctx, ethereumRPCURL)
	if err != nil {
		fmt.Println("Ethereum RPC API unavailable:", err)
		return
	}
	etherReader := etherstream.Reader{Backend: ethereumRPCClient}

	matchERC20Txs := ethereum.FilterQuery{Topics: [][]ether.Hash{{
		ether.HexToHash("0xeb6c7d1cd53bd4a9d7c4478386be075d97a6372e435e72cb37313dfa17ad00d7"),
	}}}
	stream, sub, history, err := etherReader.QueryWithHistory(ctx, &matchERC20Txs)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer sub.Unsubscribe()

	if len(history) != 0 {
		fmt.Println("historic entries ✓")
	}
	fmt.Print(cap(stream), "-slot stream buffer ✓\n")
	// Output:
	// historic entries ✓
	// 60-slot stream buffer ✓
}
