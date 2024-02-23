package openfile

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ReadSectors(sectorFile string) ([]uint64, error) {
	file, err := os.Open(sectorFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	var sectors []uint64
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