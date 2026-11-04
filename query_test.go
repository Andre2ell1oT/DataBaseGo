package main

import "testing"

func TestQueryLifecycle(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, query := range []string{
		"CREATE_TABLE users id name",
		"INSERT users Alice",
		"UPDATE users 1 Bob",
		"SELECT users 1",
		"SELECT users",
		"DELETE users 1",
	} {
		if ok, message := validateQuery(query); !ok {
			t.Fatalf("validate %q: %s", query, message)
		}
		requestType, table, arguments := parseQuery(query)
		ok, result := handleQuery(requestType, table, arguments)
		if !ok {
			t.Fatalf("execute %q: %s", query, result)
		}
		if query == "SELECT users 1" && result != "1,Bob" {
			t.Fatalf("unexpected record: %q", result)
		}
	}
	if ok, _ := handleQuery(Select, "users", []string{"1"}); ok {
		t.Fatal("deleted record found")
	}
	if ok, _ := handleQuery(CreateTable, "users", []string{"id", "name"}); ok {
		t.Fatal("existing table overwritten")
	}
}

func TestQueryValidationAndParsing(t *testing.T) {
	// Проверка формата не зависит от существования файла.
	if ok, message := validateQuery("  SELECT\t missing  "); !ok {
		t.Fatal(message)
	}
	for _, query := range []string{"", "UNKNOWN users", "SELECT", "DELETE users", "SELECT users 0", "UPDATE users 1", "CREATE_TABLE users name", "SELECT ../users", "SELECT users 1 extra"} {
		if ok, _ := validateQuery(query); ok {
			t.Errorf("accepted %q", query)
		}
		requestType, table, arguments := parseQuery(query)
		if ok, _ := handleQuery(requestType, table, arguments); ok {
			t.Errorf("executed invalid query %q", query)
		}
	}
	requestType, table, arguments := parseQuery("  UPDATE\tusers  1 Bob ")
	if requestType != Update || table != "users" || len(arguments) != 2 || arguments[0] != "1" || arguments[1] != "Bob" {
		t.Fatalf("unexpected parse: %q %q %q", requestType, table, arguments)
	}
}
