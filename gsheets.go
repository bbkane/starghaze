package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"runtime"
	"time"

	"github.com/bbkane/warg/flag"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func gSheetsOpen(pf flag.PassedFlags) error {
	spreadsheetId := pf["--spreadsheet-id"].(string)
	sheetID := pf["--sheet-id"].(int)

	link := fmt.Sprintf(
		"https://docs.google.com/spreadsheets/d/%s/edit#gid=%d",
		spreadsheetId,
		sheetID,
	)
	fmt.Printf("Opening: %s\n", link)

	// https://stackoverflow.com/a/39324149/2958070
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, link)
	exec.Command(cmd, args...).Start()
	return nil
}

func gSheetsUpload(pf flag.PassedFlags) error {
	csvPath := pf["--csv-path"].(string)
	spreadsheetId := pf["--spreadsheet-id"].(string)
	sheetID := pf["--sheet-id"].(int)
	timeout := pf["--timeout"].(time.Duration)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	csvBytes, err := ioutil.ReadFile(csvPath)
	if err != nil {
		return fmt.Errorf("csv read error: %s: %w", csvPath, err)
	}
	csvStr := string(csvBytes)

	creds, err := google.FindDefaultCredentials(ctx, sheets.SpreadsheetsScope)
	if err != nil {
		return fmt.Errorf("can't find default credentials: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %w", err)
	}

	requests := []*sheets.Request{
		// Erase current cells
		// https://stackoverflow.com/q/37928947/2958070
		{
			UpdateCells: &sheets.UpdateCellsRequest{
				Fields: "*",
				Range: &sheets.GridRange{
					SheetId: 0,
				},
			},
		},
		// https://stackoverflow.com/q/42362702/2958070
		{
			PasteData: &sheets.PasteDataRequest{
				Coordinate: &sheets.GridCoordinate{
					ColumnIndex: 0,
					RowIndex:    0,
					// https://developers.google.com/sheets/api/guides/concepts
					SheetId: int64(sheetID), // TODO: prefer reading an int64 flag, not casting :)
				},
				Data:      csvStr,
				Delimiter: ",",
				Type:      "PASTE_NORMAL",
			},
		},
	}

	rb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	resp, err := srv.Spreadsheets.BatchUpdate(
		spreadsheetId,
		rb,
	).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("batch error failure: %w", err)
	}

	fmt.Printf("Status Code: %d\n", resp.HTTPStatusCode)
	return nil
}
