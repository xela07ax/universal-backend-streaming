package providers

import (
	"fmt"
	"strings"
)

// ComparableUrl — это базовая функция для формирования URL.
// Она объединяет хост и роут, следя за тем, чтобы не было дублирующихся слешей.
func ComparableUrl(host, route, params string, args interface{}) string {
	// Очищаем хост от лишнего слеша в конце
	host = strings.TrimSuffix(host, "/")

	// Очищаем роут от лишнего слеша в начале
	route = strings.TrimPrefix(route, "/")

	// Формируем базу
	fullURL := fmt.Sprintf("%s/%s", host, route)

	// Если есть параметры, добавляем их
	if params != "" {
		if !strings.HasPrefix(params, "?") {
			params = "?" + params
		}
		fullURL += params
	}

	return fullURL
}
