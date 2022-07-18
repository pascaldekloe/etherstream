[![Go Reference](https://pkg.go.dev/badge/github.com/pascaldekloe/etherstream.svg)](https://pkg.go.dev/github.com/pascaldekloe/etherstream)

EtherStream is a library for event-sourcing with Ethereum blockchains.

This is free and unencumbered software released into the
[public domain](https://creativecommons.org/publicdomain/zero/1.0).

Your feedback and code is welcome.


## Use

Fetch a live stream plus all previous entries with something like the following.

```go
ethereumRPCURL := "wss://eth-goerli.g.alchemy.com/v2/8DXx4tv3-IXWBgwrtyhhmElxV0_VM9YK"
ethereumRPCClient, err := ethclient.DialContext(ctx, ethereumRPCURL)
if err != nil {
	log.Fatal("Ethereum RPC API unavailable:", err)
}
etherReader := etherstream.Reader{Backend: ethereumRPCClient}

exampleQ := ethereum.FilterQuery{Topics: [][]ether.Hash{{
	ether.HexToHash("0xeb6c7d1cd53bd4a9d7c4478386be075d97a6372e435e72cb37313dfa17ad00d7"),
}}}
stream, sub, history, err := etherReader.QueryWithHistory(ctx, &exampleQ)
if err != nil {
	log.Print(err)
	return
}
defer sub.Unsubscribe()
```
