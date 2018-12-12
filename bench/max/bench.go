package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/json-iterator/go"
)

const (
	UnconfirmedTXsNum = "http://localhost:26657/num_unconfirmed_txs"
	PostTx            = "http://localhost:26657/broadcast_tx_async?tx="
	NRequests         = 50000

	logStep = 1000
)

var NChunks = 5 * runtime.NumCPU()

func main() {
	chunks, chunkSize := getChunks()
	startTime := time.Now()
	done := new(uint32)

	for n := 0; n < chunks; n++ {
		from := n * chunkSize

		to := (n + 1) * chunkSize
		if to > NRequests {
			to = NRequests
		}

		go postTxs(from, to, done)
	}

	// wait until all txs passed
	time.Sleep(50 * time.Millisecond)
	for !hasUnconfirmedTxs(false) {
		time.Sleep(50 * time.Millisecond)
	}

	endTime := time.Now()

	fmt.Println("Done", NRequests)
	fmt.Println("Total time", endTime.Sub(startTime))
	fmt.Println("RPS", float64(NRequests)/float64(endTime.Sub(startTime).Seconds()))
}

func getChunks() (int, int) {
	var chunks = NChunks
	if NRequests%chunks != 0 {
		chunks++
	}
	var chunkSize = NRequests / NChunks

	return chunks, chunkSize
}

func postTxs(from, to int, done *uint32) {
	fmt.Println("tx from", from, "to", to-1)
	for i := from; i < to; i++ {
		postTx(i)
		done := atomic.AddUint32(done, 1)

		if done%logStep == 0 {
			fmt.Println("Already done", done)
		}
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
	_, err = io.Copy(ioutil.Discard, resp.Body)

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
