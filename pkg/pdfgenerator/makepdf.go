package pdfgenerator

import (
	"errors"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/jung-kurt/gofpdf"
	"os"
	"time"
)

const rmPerWeek = 455.00

func resolveFile(dirPath, filename string) (string, error) {
	if dirPath == "" {
		dirPath = os.TempDir()
	}
	fi, err := os.Stat(dirPath)
	if err != nil {
		return "", err
	}

	if mode := fi.Mode(); mode.IsRegular() {
		return "", errors.New(fmt.Sprint("Path ", dirPath, " is not a valid directory."))
	}
	return dirPath + "/" + filename, nil
}

func MakePdfToDir(claimFormImageFilePath, dirPath string, entries []pagerduty.RenderedScheduleEntry, claimDate time.Time) ([]string, error) {
	type periodEntry struct {
		from, to time.Time
	}

	type periodEntries []periodEntry

	entriesByName := make(map[string]periodEntries)

	for _, entry := range entries {
		from, _ := time.Parse(time.RFC3339, entry.Start)
		to, _ := time.Parse(time.RFC3339, entry.End)
		name := entry.User.Summary

		periodEntries := entriesByName[name]
		if periodEntries == nil {
			entriesByName[name] = []periodEntry{{from, to}}
		} else {
			entriesByName[name] = append(periodEntries, periodEntry{from, to})
		}
	}

	results := []string{}
	for name, periods := range entriesByName {
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		pdf.SetFont("Helvetica", "", 10)
		_, lineHt := pdf.GetFontSize()

		pdf.Image(claimFormImageFilePath, 0, 0, 210, 0, false, "", 0, "")

		pdf.SetXY(57, 56)
		pdf.Write(lineHt, name)

		pdf.SetXY(57, 63)
		pdf.Write(lineHt, claimDate.Format("Monday, 02 Jan 2006"))

		for index, period := range periods {
			if index == 10 {
				break
			}
			offset := float64(index) * lineHt * 2.1

			pdf.SetXY(28, offset+96)
			pdf.Write(lineHt, period.from.Format("02 Jan 2006")+" to "+period.to.Format("02 Jan 2006"))

			pdf.SetXY(75, offset+96)
			pdf.Write(lineHt, "Support On-Call Claim")

			pdf.SetXY(160, offset+96)
			pdf.Write(lineHt, fmt.Sprint("RM ", rmPerWeek))
		}

		pdf.SetXY(160, 166)
		pdf.Write(lineHt, fmt.Sprint("RM ", len(periods)*rmPerWeek))

		fileStr, err := resolveFile(dirPath, periods[0].from.Format("2006-01-02")+" until "+periods[len(periods)-1].to.Format("2006-01-02")+" "+name+" support oncall claim.pdf")
		if err != nil {
			return nil, err
		}
		results = append(results, fileStr)

		if err := pdf.OutputFileAndClose(fileStr); err != nil {
			fmt.Println("Unable to generate "+fileStr+" because:", err)
		} else {
			fmt.Println("File " + fileStr + " generated.")
		}
	}
	return results, nil
}
