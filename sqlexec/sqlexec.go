package sqlexec

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strings"
)

func ReadDSN() (dsn string, err error) {
	file, err := os.Open("dsn")
	if err != nil {
		return "", fmt.Errorf("failed to open dsn file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read dsn file: %w", err)
	}

	if len(lines) != 1 {
		return "", fmt.Errorf("dsn file must contain exactly one non-empty line")
	}

	return lines[0], err
}

func InitDB(DSN string) (DB *sql.DB, err error) {

	DB, err = sql.Open("mysql", DSN)
	if err != nil {
		return nil, err
	}

	info := fmt.Sprintf("dsn check success")
	log.Println(info)

	err = DB.Ping()
	if err != nil {
		return nil, err
	}

	info = fmt.Sprintf("database connect success")
	log.Println(info)

	return DB, nil
}

func GetMiner(db *sql.DB, cluster string) (miner string, err error) {
	SQL := fmt.Sprintf("SELECT f0 FROM cluster_list WHERE name='%s'", cluster)
	err = db.QueryRow(SQL).Scan(&miner)
	return miner, err
}
