package get

import (
	"bufio"
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"os"
	"strconv"
	"strings"
)

func FromText(sectorFile string) (sectors []uint64, err error) {
	file, err := os.Open(sectorFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		sector, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			fmt.Printf("Skipping line with invalid sector: %s\n", line)
			continue
		}
		sectors = append(sectors, sector)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return sectors, nil
}

func FromChain(ctx context.Context, delegate v0api.FullNode, addr address.Address) (sectors []uint64, err error) {

	sectorsOnChain, err := delegate.StateMinerActiveSectors(ctx, addr, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	for _, sector := range sectorsOnChain {
		s := uint64(sector.SectorNumber)
		sectors = append(sectors, s)
	}

	return sectors, nil

}
