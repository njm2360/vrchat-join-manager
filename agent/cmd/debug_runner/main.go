package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"jm-client/internal/core"
)

// csvToVrcLine は GrayLog CSV のタイムスタンプ（UTC ISO 8601）を
// VRChat ログ形式に変換して message と結合する。
func csvToVrcLine(timestampUTC, message string, loc *time.Location) (string, error) {
	t, err := time.Parse(time.RFC3339, timestampUTC)
	if err != nil {
		// "Z" 末尾でない場合のフォールバック
		t, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", timestampUTC)
		if err != nil {
			return "", fmt.Errorf("parse timestamp %q: %w", timestampUTC, err)
		}
	}
	return t.In(loc).Format("2006.01.02 15:04:05") + " " + message, nil
}

func procRawLog(path string, parser *core.LogParser) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, readErr := f.Read(tmp)
		buf = append(buf, tmp[:n]...)
		for {
			idx := strings.IndexByte(string(buf), '\n')
			if idx < 0 {
				break
			}
			line := strings.TrimRight(string(buf[:idx]), "\r")
			buf = buf[idx+1:]
			parser.OnLine(path, line)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	// 末尾に改行がない場合の残り
	if len(buf) > 0 {
		parser.OnLine(path, strings.TrimRight(string(buf), "\r\n"))
	}
	return nil
}

func procCSV(path string, parser *core.LogParser, loc *time.Location) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("read CSV header: %w", err)
	}
	// カラムインデックスを解決
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[h] = i
	}
	tsCol, ok1 := colIdx["timestamp"]
	msgCol, ok2 := colIdx["message"]
	if !ok1 || !ok2 {
		return fmt.Errorf("CSV must have 'timestamp' and 'message' columns")
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read CSV row: %w", err)
		}
		line, err := csvToVrcLine(row[tsCol], row[msgCol], loc)
		if err != nil {
			log.Printf("skip row: %v", err)
			continue
		}
		parser.OnLine(path, line)
	}
	return nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <log_file_or_csv>\n", os.Args[0])
		os.Exit(1)
	}
	target := os.Args[1]

	// .env 読み込み
	exe, _ := os.Executable()
	if err := godotenv.Load(".env"); err != nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}

	baseURL := os.Getenv("API_BASE")
	if baseURL == "" {
		log.Fatal("API_BASE environment variable not set")
	}

	tzName := os.Getenv("LOG_TZ")
	if tzName == "" {
		log.Fatal("LOG_TZ environment variable not set")
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Fatalf("LOG_TZ: %v", err)
	}

	parser := core.NewLogParser(core.NewApiClient(baseURL), loc)

	var procErr error
	if strings.EqualFold(filepath.Ext(target), ".csv") {
		procErr = procCSV(target, parser, loc)
	} else {
		procErr = procRawLog(target, parser)
	}
	if procErr != nil {
		log.Fatalf("error: %v", procErr)
	}
}
