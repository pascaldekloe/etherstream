package etherstream

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	// lots of poor naming in go-ethereum 👾
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ether "github.com/ethereum/go-ethereum/common"
	chain "github.com/ethereum/go-ethereum/core/types"
)

func TestEventsWithHistory(t *testing.T) {
	sigHash := ether.HexToHash("0xbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeef")
	stub := logStub{
		historicN: 1,
		liveN:     1,
		overlapN:  1,
		wantQuery: ethereum.FilterQuery{
			Topics: [][]ether.Hash{{sigHash}},
		},
	}

	live, sub, history, err := Reader{Backend: &stub}.EventsWithHistory(&abi.Event{ID: sigHash})
	if err != nil {
		t.Fatal("got error:", err)
	}
	if len(history) != stub.historicN {
		t.Errorf("got %d historic entries, want %d", len(history), stub.historicN)
	}
	if _, ok := sub.(subscriptionMock); !ok {
		t.Errorf("got unknown subscription type %T, want etherstream.subscriptionMock", sub)
	}

	// live entry overlaps with historic one
	select {
	case l := <-live:
		t.Errorf("got live entry %+v, want none", l)
	default:
		break // OK
	}
}

func TestNoLive(t *testing.T) {
	stub := logStub{
		historicN: 99,
		liveN:     0,
		overlapN:  0,
		wantQuery: ethereum.FilterQuery{
			ToBlock: big.NewInt(1001),
		},
	}

	live, _, history, err := Reader{Backend: &stub}.QueryWithHistory(&stub.wantQuery)
	if err != nil {
		t.Fatal("got error:", err)
	}
	if len(history) != stub.historicN {
		t.Errorf("got %d historic entries, want %d", len(history), stub.historicN)
	}

	select {
	case l := <-live:
		t.Errorf("got live entry %+v, want none", l)
	default:
		// live entry overlaps with historic one
		break // OK
	}
}

func TestNoHistory(t *testing.T) {
	stub := logStub{
		historicN: 0,
		liveN:     2,
		overlapN:  0,
		wantQuery: ethereum.FilterQuery{
			FromBlock: big.NewInt(1001),
		},
	}

	live, _, history, err := Reader{Backend: &stub}.QueryWithHistory(&stub.wantQuery)
	if err != nil {
		t.Fatal("got error:", err)
	}
	if len(history) != 0 {
		t.Errorf("got %d historic entries, want none", len(history))
	}

	timeout := time.NewTimer(100 * time.Millisecond)
	defer timeout.Stop()
	select {
	case l := <-live:
		if l.BlockNumber != 0 {
			t.Errorf("got block number %d, want 0 (for first)", l.BlockNumber)
		}
	case <-timeout.C:
		t.Fatal("live entry reception timeout")
	}
	select {
	case l := <-live:
		if l.BlockNumber != 1 {
			t.Errorf("got block number %d, want 1 (for second)", l.BlockNumber)
		}
	case <-timeout.C:
		t.Fatal("live entry reception timeout")
	}
}

func TestReaderNoOverlap(t *testing.T) {
	sigHash := ether.HexToHash("0xbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeef")
	r := Reader{Backend: &logStub{
		historicN: 1,
		liveN:     1,
		overlapN:  0,
		wantQuery: ethereum.FilterQuery{
			Topics: [][]ether.Hash{{sigHash}},
		},
	}}

	_, _, _, err := r.EventsWithHistory(&abi.Event{ID: sigHash})
	if !errors.Is(err, errNoOverlap) {
		t.Errorf("got error %v, want a %q", err, errNoOverlap)
	}
}

type logStub struct {
	historicN int
	liveN     int
	overlapN  int
	wantQuery ethereum.FilterQuery
}

func (stub *logStub) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]chain.Log, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(q, stub.wantQuery) {
		return nil, fmt.Errorf("FilterLogs got query %+v, want %+v", q, stub.wantQuery)
	}

	logs := make([]chain.Log, stub.historicN)
	for i := range logs {
		logs[i].BlockNumber = uint64(i)
	}
	return logs, nil
}

func (stub *logStub) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- chain.Log) (ethereum.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(q, stub.wantQuery) {
		return nil, fmt.Errorf("SubscribeFilterLogs got query %+v, want %+v", q, stub.wantQuery)
	}

	go func() {
		for i := 0; i < stub.liveN; i++ {
			if stub.historicN < stub.overlapN {
				panic("irrational test parameter")
			}

			// historic starts counting from 0
			seqNo := i + stub.historicN - stub.overlapN
			ch <- chain.Log{BlockNumber: uint64(seqNo)}
		}
	}()
	return subscriptionMock{}, nil
}

type subscriptionMock struct{}

func (mock subscriptionMock) Unsubscribe()      {}
func (mock subscriptionMock) Err() <-chan error { return nil }

func TestEventOrder(t *testing.T) {
	var tests = []struct {
		a, b      chain.Log
		wantOrder int
	}{
		{
			a:         chain.Log{BlockNumber: 98, TxIndex: 5},
			b:         chain.Log{BlockNumber: 99, TxIndex: 3},
			wantOrder: 1,
		},
		{
			a:         chain.Log{BlockNumber: 99, TxIndex: 3},
			b:         chain.Log{BlockNumber: 98, TxIndex: 5},
			wantOrder: -1,
		},
		{
			a:         chain.Log{BlockNumber: 99, TxIndex: 3},
			b:         chain.Log{BlockNumber: 99, TxIndex: 5},
			wantOrder: 2,
		},
		{
			a:         chain.Log{BlockNumber: 99, TxIndex: 5},
			b:         chain.Log{BlockNumber: 99, TxIndex: 3},
			wantOrder: -2,
		},
		{
			a:         chain.Log{BlockNumber: 99, TxIndex: 3},
			b:         chain.Log{BlockNumber: 99, TxIndex: 3},
			wantOrder: 0,
		},

		{
			a:         chain.Log{TxHash: ether.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
			b:         chain.Log{TxHash: ether.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")},
			wantOrder: 1,
		},
	}

	for _, test := range tests {
		got := Order(&test.a, &test.b)
		if got != test.wantOrder {
			t.Errorf("got %d, want %d", got, test.wantOrder)
		}
	}
}
