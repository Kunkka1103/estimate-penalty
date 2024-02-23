package main

import (
	"context"
	"estimate-penalty/estimate"
	"estimate-penalty/openfile"
	"flag"
	"fmt"
	"github.com/filecoin-project/go-state-types/big"
	"golang.org/x/sync/semaphore"
	"log"
	"sync"
)

var miner = flag.String("miner", "", "miner")
var sectorFile = flag.String("sectors", "", "sectors file")
var lotusAPI = flag.String("l", "http://127.0.0.1:1234/rpc/v0", "lotusAPI")
var concurrentLimit = flag.Int("concurrency", 100, "The maximum number of concurrent TerminateSector operations")

func main() {

	flag.Parse()

	// 检查命令行参数
	if *miner == "" || *sectorFile == "" {
		fmt.Println("Miner address and sectors file are required")
		return
	}

	// 检查扫描过程中是否有错误发生
	sectors, err := openfile.ReadSectors(*sectorFile)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		return
	}

	// 打印slice 长度
	log.Printf("number of sectors: %d", len(sectors))
	var wg sync.WaitGroup
	penaltySum := big.Zero()
	var mu sync.Mutex // 用于保护penaltySum

	sem := semaphore.NewWeighted(int64(*concurrentLimit)) // 最大并发数为10
	ctx := context.Background()

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

			penalty, err := estimate.TerminateSector(ctx, *lotusAPI, *miner, sector)
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
