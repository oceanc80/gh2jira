package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"sigs.k8s.io/yaml"
)

// ValidateFormat rejects unknown output formats before any configuration or I/O.
func ValidateFormat(format string) error {
	switch format {
	case "json", "yaml", "table":
		return nil
	default:
		return fmt.Errorf("unsupported report output %q (use json, yaml or table)", format)
	}
}

// Write renders report rows and propagates output failures, including short writes.
func Write(w io.Writer, format string, rows []Row) error {
	if err := ValidateFormat(format); err != nil {
		return err
	}
	if rows == nil {
		rows = []Row{}
	}
	var data []byte
	if format == "table" {
		var buf bytes.Buffer
		tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "JIRA KEY\tJIRA STATUS\tGITHUB REPOSITORY\tTYPE\tNUMBER\tGITHUB STATE\tMERGED\tGITHUB URL\tLINK STATUS\tPROBLEM")
		for _, row := range rows {
			var source Source
			if row.GitHub != nil {
				source = *row.GitHub
			}
			number, merged := "", ""
			if source.Number != 0 {
				number = strconv.Itoa(source.Number)
			}
			if source.Merged != nil {
				merged = strconv.FormatBool(*source.Merged)
			}
			// Quote free-text states so tabs/newlines cannot create misleading rows.
			fmt.Fprintf(tw, "%s\t%q\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				row.Jira.Key, row.Jira.Status, source.Repository, source.Type, number,
				source.State, merged, source.URL, row.LinkStatus, row.Problem)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		data = buf.Bytes()
	} else {
		var err error
		data, err = json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		if format == "yaml" {
			data, err = yaml.JSONToYAML(data)
			if err != nil {
				return err
			}
		} else {
			data = append(data, '\n')
		}
	}
	n, err := w.Write(data)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("write report: %w", io.ErrShortWrite)
	}
	return nil
}
