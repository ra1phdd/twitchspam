package aliases

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

func (a *Aliases) randint(args string) string {
	parts := strings.Fields(args)
	if len(parts) != 2 {
		return "{неверные аргументы}"
	}

	min, err1 := strconv.Atoi(a.ReplacePlaceholders(parts[0], nil))
	max, err2 := strconv.Atoi(a.ReplacePlaceholders(parts[1], nil))
	if err1 != nil || err2 != nil {
		return "{ошибка преобразования в число}"
	}

	if min > max {
		min, max = max, min
	}

	num := rand.Intn(max-min+1) + min
	return fmt.Sprintf("%d", num)
}
