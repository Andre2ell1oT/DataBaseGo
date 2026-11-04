package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	CreateTable = "CREATE_TABLE"
	Select      = "SELECT"
	Insert      = "INSERT"
	Update      = "UPDATE"
	Delete      = "DELETE"
)

func validateQuery(query string) (bool, string) {
	// Проверяет валидность пользовательского запроса.
	// Возвращает true и пустую строку, если запрос валиден, иначе false и сообщение об ошибке.
	requestType, table, arguments := parseQuery(query)
	return validateQueryParts(requestType, table, arguments)
}

// Проверяет только формат команды, без обращения к файлам.
func validateQueryParts(requestType string, table string, arguments []string) (bool, string) {
	if requestType == "" {
		return false, "Пустой запрос."
	}

	switch requestType {
	case CreateTable, Select, Insert, Update, Delete:
	default:
		return false, fmt.Sprintf("Неизвестная команда. Доступны: %s, %s, %s, %s, %s.", CreateTable, Select, Insert, Update, Delete)
	}

	if table == "" {
		return false, "Не указано имя таблицы."
	}
	if table == "." || table == ".." || strings.ContainsAny(table, "/\\\x00") {
		return false, "Имя таблицы не должно содержать путь к файлу."
	}

	switch requestType {
	case CreateTable:
		if len(arguments) < 1 {
			return false, "Формат: CREATE_TABLE имя_таблицы id [колонки...]."
		}
		if !validHeaders(arguments) {
			return false, "Первая колонка должна называться id; названия колонок не должны быть пустыми или повторяться."
		}
	case Select:
		if len(arguments) > 1 {
			return false, "Формат: SELECT имя_таблицы [ID]."
		}
	case Insert:
		if len(arguments) < 1 {
			return false, "Формат: INSERT имя_таблицы значение [значения...]."
		}
	case Update:
		if len(arguments) < 2 {
			return false, "Формат: UPDATE имя_таблицы ID значение [значения...]."
		}
	case Delete:
		if len(arguments) != 1 {
			return false, "Формат: DELETE имя_таблицы ID."
		}
	}

	if requestType == Update || requestType == Delete || (requestType == Select && len(arguments) == 1) {
		id, err := strconv.Atoi(arguments[0])
		if err != nil || id < 1 || strconv.Itoa(id) != arguments[0] {
			return false, "ID должен быть положительным целым числом без знака и ведущих нулей."
		}
	}

	return true, ""
}

func parseQuery(query string) (string, string, []string) {
	// Fields пропускает лишние пробелы и табуляцию.
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return "", "", nil
	}
	if len(parts) == 1 {
		return parts[0], "", nil
	}
	return parts[0], parts[1], parts[2:]
}

func handleQuery(requestType string, table string, arguments []string) (bool, string) {
	// Проверка нужна и при прямом вызове handleQuery без validateQuery.
	if ok, message := validateQueryParts(requestType, table, arguments); !ok {
		return false, message
	}

	info, err := os.Stat(table + ".csv")
	if requestType == CreateTable {
		if err == nil {
			return false, "Таблица уже существует."
		}
		if !os.IsNotExist(err) {
			return false, "Не удалось проверить наличие таблицы."
		}
	} else {
		if os.IsNotExist(err) {
			return false, "Таблица не найдена."
		}
		if err != nil {
			return false, "Не удалось проверить наличие таблицы."
		}
		if !info.Mode().IsRegular() {
			return false, "Таблица должна быть обычным файлом."
		}
	}

	switch requestType {
	case CreateTable:
		if !createTable(table, arguments) {
			return false, "Не удалось создать таблицу."
		}
		return true, "Таблица создана."
	case Select:
		if len(arguments) == 0 {
			data, ok := selectAll(table)
			if !ok {
				return false, "Не удалось прочитать таблицу."
			}
			return true, data
		}
		data, ok := selectRecord(table, arguments[0])
		if !ok {
			return false, "Запись не найдена или таблицу не удалось прочитать."
		}
		return true, data
	case Insert:
		if !insertRecord(table, arguments) {
			return false, "Не удалось добавить запись. Проверьте значения полей и содержимое таблицы."
		}
		return true, "Запись добавлена."
	case Update:
		if !updateRecord(table, arguments[0], arguments[1:]) {
			return false, "Не удалось обновить запись. Проверьте ID, значения полей и содержимое таблицы."
		}
		return true, "Запись обновлена."
	case Delete:
		if !deleteRecord(table, arguments[0]) {
			return false, "Не удалось удалить запись. Проверьте ID и содержимое таблицы."
		}
		return true, "Запись удалена."
	}
	return false, "Неизвестная команда."
}
