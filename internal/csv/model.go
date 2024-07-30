package csv

import "fmt"

const cfFieldSize = 10

func ValidateCF(header []string, bodies [][]string) error {
	if len(header) != cfFieldSize {
		return fmt.Errorf("invalid field size: header")
	}
	for i, b := range bodies {
		if len(b) != cfFieldSize {
			return fmt.Errorf("invalid field size: bodies #%d", i+1)
		}
	}
	return nil
}
