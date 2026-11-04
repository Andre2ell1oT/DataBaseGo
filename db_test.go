package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestTableLifecycle(t *testing.T) {
	table := t.TempDir() + "/users"
	if !createTable(table, []string{"id", "name", "age"}) {
		t.Fatal("create failed")
	}
	if id, ok := getNextID(table); !ok || id != 1 {
		t.Fatalf("empty table next ID: (%d, %v)", id, ok)
	}
	if !insertRecord(table, []string{"Мария, \"Маша\"\nИванова", "30"}) {
		t.Fatal("insert failed")
	}
	text, ok := selectRecord(table, "1")
	if !ok {
		t.Fatal("select failed")
	}
	record, err := csv.NewReader(strings.NewReader(text)).Read()
	if err != nil || len(record) != 3 || record[1] != "Мария, \"Маша\"\nИванова" {
		t.Fatalf("CSV round trip: %q, %v", record, err)
	}
	before, _ := selectAll(table)
	if createTable(table, []string{"id", "other"}) || insertRecord(table, []string{"missing age"}) || updateRecord(table, "1", []string{"missing age"}) {
		t.Fatal("invalid operation succeeded")
	}
	after, _ := selectAll(table)
	if before != after {
		t.Fatal("invalid operation changed table")
	}
	if !updateRecord(table, "1", []string{"Аня", "2"}) {
		t.Fatal("update failed")
	}
	if got, ok := selectRecord(table, "1"); !ok || got != "1,Аня,2" {
		t.Fatalf("updated record: (%q, %v)", got, ok)
	}
	if updateRecord(table, "99", []string{"нет", "0"}) || deleteRecord(table, "99") {
		t.Fatal("missing ID operation succeeded")
	}
	if !deleteRecord(table, "1") {
		t.Fatal("delete failed")
	}
	if got, ok := selectAll(table); !ok || got != "id,name,age\n" {
		t.Fatalf("table after deletion: (%q, %v)", got, ok)
	}
}

func TestExistingCSV(t *testing.T) {
	table := t.TempDir() + "/users"
	if err := os.WriteFile(table+".csv", []byte("id,name\n5,Alice\n2,Bob"), 0640); err != nil {
		t.Fatal(err)
	}
	if id, ok := getNextID(table); !ok || id != 6 {
		t.Fatalf("next ID: (%d, %v)", id, ok)
	}
	if !insertRecord(table, []string{"Carol"}) {
		t.Fatal("insert into CSV without final newline failed")
	}
	if got, ok := selectRecord(table, "6"); !ok || got != "6,Carol" {
		t.Fatalf("inserted record: (%q, %v)", got, ok)
	}
	info, err := os.Stat(table + ".csv")
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatal("file permissions changed", err)
	}
}

func TestInvalidTables(t *testing.T) {
	for _, data := range []string{"", "name\nAlice\n", "id,name\n1\n", "id,name\nx,Alice\n", "id,name\n1,Alice\n1,Bob\n", "id,name\n0,Alice\n", "id,name\n" + strconv.Itoa(int(^uint(0)>>1)) + ",Alice\n"} {
		table := t.TempDir() + "/users"
		if err := os.WriteFile(table+".csv", []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, ok := getNextID(table); ok {
			t.Fatalf("accepted invalid or exhausted table: %q", data)
		}
		if insertRecord(table, []string{"Bob"}) {
			t.Fatal("insert into invalid table succeeded")
		}
		after, err := os.ReadFile(table + ".csv")
		if err != nil || string(after) != data {
			t.Fatal("failed operation changed original file", err)
		}
	}
}

func TestConcurrentInserts(t *testing.T) {
	table := t.TempDir() + "/users"
	if !createTable(table, []string{"id", "name"}) {
		t.Fatal("create failed")
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !insertRecord(table, []string{"Alice"}) {
				t.Error("concurrent insert failed")
			}
		}()
	}
	wg.Wait()
	if id, ok := getNextID(table); !ok || id != 11 {
		t.Fatalf("next ID after concurrent inserts: (%d, %v)", id, ok)
	}
}
