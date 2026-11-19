package upload

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// sizeUnits maps a size suffix to its multiplier. The project uses binary
// multipliers, so MB and MiB both mean 1024 * 1024 bytes.
var sizeUnits = map[string]int64{
	"":    1,
	"B":   1,
	"KB":  1 << 10,
	"KIB": 1 << 10,
	"MB":  1 << 20,
	"MIB": 1 << 20,
	"GB":  1 << 30,
	"GIB": 1 << 30,
}

// ParseSize parses a human-readable byte size such as "10MB", "2MiB", or a
// plain number of bytes. Sizes must be positive and must not overflow int64.
func ParseSize(value string) (int64, error) {
	text := strings.ToUpper(strings.TrimSpace(value))
	if text == "" {
		return 0, fmt.Errorf("parse size: empty value")
	}

	split := 0
	for split < len(text) && text[split] >= '0' && text[split] <= '9' {
		split++
	}
	if split == 0 {
		return 0, fmt.Errorf("parse size %q: expected a positive number", value)
	}

	number, err := strconv.ParseInt(text[:split], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse size %q: %w", value, err)
	}
	if number <= 0 {
		return 0, fmt.Errorf("parse size %q: size must be positive", value)
	}

	unit := strings.TrimSpace(text[split:])
	multiplier, ok := sizeUnits[unit]
	if !ok {
		return 0, fmt.Errorf("parse size %q: unknown unit %q", value, unit)
	}
	if number > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("parse size %q: value overflows int64", value)
	}
	return number * multiplier, nil
}
