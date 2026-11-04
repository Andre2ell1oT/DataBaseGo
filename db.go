package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Защищает операции с таблицами от одновременного выполнения в этом процессе.
var tableMu sync.Mutex

func createTable(tableName string, fieldNames []string) bool {
	// Первый столбец таблицы всегда должен называться id.
	tableMu.Lock()
	defer tableMu.Unlock()

	if !validHeaders(fieldNames) {
		return false
	}
	file, err := os.OpenFile(tableName+".csv", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return false
	}
	writer := csv.NewWriter(file)
	writeErr := writer.WriteAll([][]string{fieldNames})
	closeErr := file.Close()
	return writeErr == nil && closeErr == nil
}

func insertRecord(tableName string, fieldValues []string) bool {
	tableMu.Lock()
	defer tableMu.Unlock()

	records, ok := readTable(tableName)
	if !ok || len(fieldValues) != len(records[0])-1 {
		return false
	}
	nextID, ok := nextRecordID(records)
	if !ok {
		return false
	}
	record := append([]string{strconv.Itoa(nextID)}, fieldValues...)
	// Перезапись также корректно обрабатывает CSV без завершающего переноса строки.
	return writeTable(tableName, append(records, record))
}

func getNextID(tableName string) (int, bool) {
	tableMu.Lock()
	defer tableMu.Unlock()

	records, ok := readTable(tableName)
	if !ok {
		return 0, false
	}
	return nextRecordID(records)
}

func nextRecordID(records [][]string) (int, bool) {
	maxID := 0
	for _, record := range records[1:] {
		id, err := strconv.Atoi(record[0])
		if err != nil || id < 1 {
			return 0, false
		}
		if id > maxID {
			maxID = id
		}
	}
	if maxID == int(^uint(0)>>1) {
		return 0, false // Следующий ID не помещается в int.
	}
	return maxID + 1, true
}

func updateRecord(tableName string, id string, fieldValues []string) bool {
	tableMu.Lock()
	defer tableMu.Unlock()

	records, ok := readTable(tableName)
	if !ok || len(fieldValues) != len(records[0])-1 {
		return false
	}
	for i := 1; i < len(records); i++ {
		if records[i][0] == id {
			records[i] = append([]string{id}, fieldValues...)
			return writeTable(tableName, records)
		}
	}
	return false
}

func selectAll(tableName string) (string, bool) {
	tableMu.Lock()
	defer tableMu.Unlock()

	data, err := os.ReadFile(tableName + ".csv")
	if err != nil {
		return "", false
	}
	return string(data), true
}

func selectRecord(tableName string, id string) (string, bool) {
	tableMu.Lock()
	defer tableMu.Unlock()

	records, ok := readTable(tableName)
	if !ok {
		return "", false
	}
	for _, record := range records[1:] {
		if record[0] == id {
			var result strings.Builder
			writer := csv.NewWriter(&result)
			if err := writer.WriteAll([][]string{record}); err != nil {
				return "", false
			}
			return strings.TrimSuffix(result.String(), "\n"), true
		}
	}
	return "", false
}

func deleteRecord(tableName string, id string) bool {
	tableMu.Lock()
	defer tableMu.Unlock()

	records, ok := readTable(tableName)
	if !ok {
		return false
	}
	for i := 1; i < len(records); i++ {
		if records[i][0] == id {
			records = append(records[:i], records[i+1:]...)
			return writeTable(tableName, records)
		}
	}
	return false
}

func validHeaders(headers []string) bool {
	if len(headers) == 0 || headers[0] != "id" {
		return false
	}
	seen := make(map[string]bool)
	for _, name := range headers {
		if strings.TrimSpace(name) == "" || seen[name] {
			return false
		}
		seen[name] = true
	}
	return true
}

// readTable и writeTable вызываются только при удержании tableMu.
func readTable(tableName string) ([][]string, bool) {
	file, err := os.Open(tableName + ".csv")
	if err != nil {
		return nil, false
	}
	records, readErr := csv.NewReader(file).ReadAll()
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(records) == 0 || !validHeaders(records[0]) {
		return nil, false
	}
	seen := make(map[int]bool)
	for _, record := range records[1:] {
		if len(record) != len(records[0]) {
			return nil, false
		}
		id, err := strconv.Atoi(record[0])
		if err != nil || id < 1 || seen[id] || strconv.Itoa(id) != record[0] {
			return nil, false
		}
		seen[id] = true
	}
	return records, true
}

func writeTable(tableName string, records [][]string) bool {
	filename := tableName + ".csv"
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	file, err := os.CreateTemp(filepath.Dir(filename), ".table-*.csv")
	if err != nil {
		return false
	}
	defer os.Remove(file.Name())

	writer := csv.NewWriter(file)
	writeErr := writer.WriteAll(records)
	if writeErr == nil {
		writeErr = file.Chmod(info.Mode().Perm())
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return false
	}
	return os.Rename(file.Name(), filename) == nil
}
