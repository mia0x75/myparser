package mysql

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode"
)

func formatENUS(number string, precision string) (string, error) {
	var buffer bytes.Buffer
	if len(precision) == 0 {
		precision = "0"
	}
	if unicode.IsDigit(rune(precision[0])) {
		for i, v := range precision {
			if unicode.IsDigit(v) {
				continue
			}
			precision = precision[:i]
			break
		}
	} else {
		precision = "0"
	}
	position, err := parsePrecision(precision)
	if err != nil {
		return "", err
	}
	if number == "" || number == "-" {
		number = "0"
	}
	if strings.HasPrefix(number, "-.") {
		number = strings.Replace(number, "-", "-0", 1)
	} else if number[0] == '.' {
		number = strings.Replace(number, ".", "0.", 1)
	}

	if (number[:1] == "-" && !unicode.IsDigit(rune(number[1]))) ||
		(!unicode.IsDigit(rune(number[0])) && number[:1] != "-") {
		buffer.Write([]byte{'0'})
		if position > 0 {
			buffer.Write([]byte{'.'})
			buffer.WriteString(strings.Repeat("0", position))
		}
		return buffer.String(), nil
	} else if number[:1] == "-" {
		buffer.Write([]byte{'-'})
		number = number[1:]
	}

	for i, v := range number {
		if unicode.IsDigit(v) {
			continue
		} else if i == 1 && number[1] == '.' {
			continue
		} else if v == '.' && number[1] != '.' {
			continue
		} else {
			number = number[:i]
			break
		}
	}

	comma := []byte{','}
	parts := strings.Split(number, ".")
	pos := 0
	if len(parts[0])%3 != 0 {
		pos += len(parts[0]) % 3
		buffer.WriteString(parts[0][:pos])
		buffer.Write(comma)
	}
	for ; pos < len(parts[0]); pos += 3 {
		buffer.WriteString(parts[0][pos : pos+3])
		buffer.Write(comma)
	}
	buffer.Truncate(buffer.Len() - 1)

	if position > 0 {
		buffer.Write([]byte{'.'})
		if len(parts) == 2 {
			if len(parts[1]) >= position {
				buffer.WriteString(parts[1][:position])
			} else {
				buffer.WriteString(parts[1])
				buffer.WriteString(strings.Repeat("0", position-len(parts[1])))
			}
		} else {
			buffer.WriteString(strings.Repeat("0", position))
		}
	}

	return buffer.String(), nil
}

func parsePrecision(precision string) (int, error) {
	value, err := strconv.ParseUint(precision, 10, 64)
	if err != nil {
		return 0, err
	}
	// Bound output allocation as well as the architecture-sized conversion.
	if value > uint64(math.MaxInt) || value > 1_000_000 {
		return 0, strconv.ErrRange
	}
	return int(value), nil
}

func formatNotSupport(number string, precision string) (string, error) {
	return "", errors.New("not support for the specific locale")
}
