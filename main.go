package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(">> ")

		if !scanner.Scan() {
			break
		}

		userQuery := scanner.Text()

		if isValid, message := validateQuery(userQuery); !isValid {
			fmt.Println("Invalid request: " + message)
			continue
		}

		requestType, table, arguments := parseQuery(userQuery)

		ok, response := handleQuery(requestType, table, arguments)
		if !ok {
			fmt.Println("Error: " + response)
		} else {
			fmt.Println(response)
		}

		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Input error:", err)
	}
}
