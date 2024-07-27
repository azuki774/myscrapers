package csv

import (
	"encoding/csv"
	"os"
)

func WriteFile(outputFile string, header []string, bodies [][]string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	// header + bodies
	var writeData [][]string

	writeData = append(writeData, header)
	writeData = append(writeData, bodies...)

	w := csv.NewWriter(f)
	return w.WriteAll(writeData)
}
