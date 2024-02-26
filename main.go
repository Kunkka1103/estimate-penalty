package main

import (
	"context"
	"estimate-penalty/estimate"
	"estimate-penalty/get"
	"estimate-penalty/sqlexec"
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

var miner = flag.String("m", "", "miner")
var clusterName = flag.String("n", "", "cluster name,example:xc64,hk01")
var sectorFile = flag.String("f", "", "sectors file")
var lotusAPI = flag.String("l", "http://127.0.0.1:1234/rpc/v0", "lotusAPI")
var concurrentLimit = flag.Int("c", 100, "最大并发数")

func main() {

	flag.Parse()

	// 检查命令行参数
	if (*miner != "" && *clusterName != "") || (*miner == "" && *clusterName == "") {
		fmt.Println("Error: Please provide either Miner or Cluster, but not both or neither.")
		return
	}

	//addr, err := address.NewFromString(*miner)
	//if err != nil {
	//	fmt.Println("Address transfer failed, please check the miner number")
	//	return
	//}

	var addr address.Address
	var err error
	if *miner != "" {
		fmt.Println("检测到你本次使用的是矿工号，推荐使用集群代号查询，可通过-h 查询使用帮助")
		addr, err = address.NewFromString(*miner)
		if err != nil {
			log.Fatalf("convert miner to addr failed,err:%s", err)
		}
	} else {

		dsn, err := sqlexec.ReadDSN()
		if err != nil {
			log.Fatalln(err)
		}
		db, err := sqlexec.InitDB(dsn)
		if err != nil {
			log.Fatalf("connect to ops db failed,err:%s", err)
		}

		m, err := sqlexec.GetMiner(db, *clusterName)
		if err != nil {
			log.Fatalf("Failed to query miner, please confirm whether the cluster is correct")
		}
		addr, err = address.NewFromString(m)
		if err != nil {
			log.Fatalf("convert miner to addr failed,err:%s", err)
		}
		fmt.Printf("cluster: %s,miner: %s,正在查询中，请稍等...\n", *clusterName, m)
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
