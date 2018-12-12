package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/json-iterator/go"
)

const (
	UnconfirmedTXsNum = "http://localhost:26657/num_unconfirmed_txs"
	PostTx            = "http://localhost:26657/broadcast_tx_async?tx="
	RPS               = 4000
)

var TxTime = time.Duration(big.NewInt(0).Div(big.NewInt(int64(time.Second)), big.NewInt(RPS)).Int64()) // ns

func main() {
	hourTimer := time.NewTimer(10*time.Hour)
	defer hourTimer.Stop()

	round := time.NewTicker(TxTime)
	defer round.Stop()

	i := 0
	var totalTime time.Duration
	mainTime := time.Now()

mainLoop:
	for {
		startTime := time.Now()

		select {
		case <-hourTimer.C:
			break mainLoop
		case <-round.C:
			postTxs(i, i+1)
		}

		endTime := time.Now()

		roundTime := endTime.Sub(startTime)
		totalTime += roundTime
		currentDuration := endTime.Sub(mainTime)

		if i%RPS == 0 {
			freq := big.NewRat(int64(i+1), int64(currentDuration))
			rps, _ := freq.Mul(freq, big.NewRat(int64(time.Second), 1)).Float64()

			fmt.Printf("Total time for round %v: %v. Total test duration %v. RPS: %v\n",
				i, roundTime, currentDuration, rps)
		}

		if i%(RPS*10) == 0 {
			hasUnconfirmedTxs(true)
		}

		i++
	}

	// wait until all txs passed
	time.Sleep(50 * time.Millisecond)
	for !hasUnconfirmedTxs(false) {
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("Done", i)
	fmt.Println("Total time", totalTime)
}

func postTxs(from, to int) {
	for i := from; i < to; i++ {
		postTx(i)
	}
}

func postTx(n int) {
	doRequest(PostTx + "\"" + strconv.Itoa(time.Now().Nanosecond()) + strconv.Itoa(n) + "\"")
}

func hasUnconfirmedTxs(withLog bool) bool {
	res := doRequest(UnconfirmedTXsNum)

	resp := new(RPCResponse)
	resp.Decode(res)

	if withLog {
		fmt.Println("Has Unconfirmed Txs", string(res))
	}

	n, err := strconv.Atoi(resp.Res.N)
	if err != nil {
		fmt.Printf("error while getting unconfirmed TXs: %v, %q\n", err, string(res))
		return true
	}

	return n == 0
}

func doRequest(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("error while http.get", err)
		return nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error while reading response body", err)
		return nil
	}

	return body
}

type RPCResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Res     Result `json:"result"`
}

type Result struct {
	N   string `json:"n_txs"`
	Txs *uint  `json:"txs"`
}

func (r *RPCResponse) Decode(input []byte) {
	var json = jsoniter.ConfigFastest
	json.Unmarshal(input, r)
}
