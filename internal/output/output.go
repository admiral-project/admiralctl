// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

var defaultOut io.Writer = os.Stdout

func SetOut(w io.Writer) {
	defaultOut = w
}

func PrintJSON(data interface{}) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(defaultOut, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Fprintln(defaultOut, string(bytes))
}

func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(defaultOut, 0, 0, 3, ' ', 0)

	// Print headers
	for i, h := range headers {
		_, _ = fmt.Fprint(w, h)
		if i < len(headers)-1 {
			_, _ = fmt.Fprint(w, "\t")
		}
	}
	_, _ = fmt.Fprintln(w)

	// Print separator
	for i := range headers {
		_, _ = fmt.Fprint(w, "---")
		if i < len(headers)-1 {
			_, _ = fmt.Fprint(w, "\t")
		}
	}
	_, _ = fmt.Fprintln(w)

	// Print rows
	for _, row := range rows {
		for i, val := range row {
			_, _ = fmt.Fprint(w, val)
			if i < len(row)-1 {
				_, _ = fmt.Fprint(w, "\t")
			}
		}
		_, _ = fmt.Fprintln(w)
	}
	_ = w.Flush()
}
