package main

import (
	"context"
	"estimate-penalty/estimate"
	"estimate-penalty/get"
	"flag"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"golang.org/x/sync/semaphore"
	"log"
	"net/http"
	"sync"
)

var miner = flag.String("m", "f0709366", "miner")
var sectorFile = flag.String("f", "", "sectors file")
var lotusAPI = flag.String("l", "http://112.124.1.253:1234/rpc/v0", "lotusAPI")
var concurrentLimit = flag.Int("c", 100, "最大并发数")

func main() {

	flag.Parse()

	// 检查命令行参数
	if *miner == "" {
		fmt.Println("Miner address is required")
		return
	}

	addr, err := address.NewFromString(*miner)
	if err != nil {
		fmt.Println("Address transfer failed, please check the miner number")
		return
	}

	ctx := context.Background()

	delegate, closer, err := ConnectClient(*lotusAPI)
	if err != nil {
		fmt.Printf("connect to lotusAPI failed, err: %s", err)
		return
	}
	defer closer()

	var sectors []uint64
	if *sectorFile == "" {
		fmt.Println("未检测到sectorFile，将计算该集群所有sector")

		sectors, err = get.FromChain(ctx, delegate, addr)
		if err != nil {
			fmt.Printf("Failed to get sector info: %v\n", err)
			return
		}
	} else {
		fmt.Println("正在读取文本")

		sectors, err = get.FromText(*sectorFile)
		if err != nil {
			fmt.Printf("Failed to read file: %v\n", err)
			return
		}
	}

	// 打印slice 长度
	log.Printf("number of sectors: %d", len(sectors))
	var wg sync.WaitGroup
	penaltySum := big.Zero()
	var mu sync.Mutex // 用于保护penaltySum

	sem := semaphore.NewWeighted(int64(*concurrentLimit))

	for _, sector := range sectors {
		wg.Add(1)
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Failed to acquire semaphore: %v\n", err)
			continue
		}

		go func(sector uint64) {
			defer wg.Done()
			defer sem.Release(1)
			// 创建新的子上下文，如果需要，可以设置超时
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			penalty, err := estimate.TerminateSector(ctx, delegate, addr, sector)
			if err != nil {
				fmt.Printf("Error terminating sector %d: %v\n", sector, err)
				return
			}
			mu.Lock()
			penaltySum = big.Add(penaltySum, penalty)
			mu.Unlock()
		}(sector)
	}

	wg.Wait()

	fmt.Printf("Total estimated penalty: %s attoFIL\n", penaltySum)
}

func ConnectClient(apiUrl string) (v0api.FullNode, jsonrpc.ClientCloser, error) {
	header := http.Header{}
	ctx := context.Background()
	return client.NewFullNodeRPCV0(ctx, apiUrl, header)
}
