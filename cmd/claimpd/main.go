package main

import (
	"errors"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/jung-kurt/gofpdf"
	"github.com/karrick/tparse"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"time"
)

const globalUsage = `🍤 Claim  - PagerDuty 🍤

Will generate claim forms for the past month from this date.

Maybe run one of these?

	$ claimpd -p 'xxx' -s 'orion' --since 'now-3mo' --until 'now+1d'
	$ claimpd -p 'xxx' -s 'orion' --since 'now-1w'
    $ claimpd -p 'xxx' -s 'orion'

`

const templateName = "template.png"
const rmPerWeek = 455.00

var authtoken string
var scheduleName string
var tparseSinceQuery string
var tparseUntilQuery string

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claimpd",
		Short: "Populate info on claim form from PagerDuty",
		Long:  globalUsage,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if str, err := cmd.Flags().GetString("pd-api-key"); err != nil || str == "" {
				return errors.New("pd-api-key not set")
			}
			if str, err := cmd.Flags().GetString("schedule-name"); err != nil || str == "" {
				return errors.New("schedule-name not set")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := downloadPdfTemplate(); err != nil {
				return err
			}

			entries, _ := fetchScheduleEntries(tparseSinceQuery, tparseUntilQuery, scheduleName)

			_, err := makePdf(entries)
			return err
		},
	}

	cmd.Flags().StringVarP(&authtoken, "pd-api-key", "p", "", "PagerDuty v2 API Key")
	cmd.Flags().StringVarP(&scheduleName, "schedule-name", "s", "", "PagerDuty v2 Schedule Name for Query")
	cmd.Flags().StringVarP(&tparseSinceQuery, "since", "", "now-1mo", "Generate report since when? Example: now-2w or now-3mo")
	cmd.Flags().StringVarP(&tparseUntilQuery, "until", "", "now", "Generate report until when? Example: now-2w or now-3mo ")

	return cmd
}

func downloadPdfTemplate() error {
	filepath := templateName
	url := "https://user-images.githubusercontent.com/898384/54095034-e4fc0380-43df-11e9-8ad2-c263b3a71c71.png"

	if _, err := os.Stat(filepath); err == nil {
		return nil
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func fetchScheduleEntries(tparseSinceQuery, tparseUntilQuery, scheduleQuery string) ([]pagerduty.RenderedScheduleEntry, error) {
	client := pagerduty.NewClient(authtoken)

	res, err := client.ListSchedules(pagerduty.ListSchedulesOptions{Query: scheduleQuery})
	if err != nil {
		return nil, err
	}

	var pdScheduleId string
	for _, schedule := range res.Schedules {
		pdScheduleId = schedule.ID
		break
	}

	since, err := tparse.ParseNow(time.RFC3339, tparseSinceQuery)
	if err != nil {
		return nil, err
	}

	until, err := tparse.ParseNow(time.RFC3339, tparseUntilQuery)
	if err != nil {
		return nil, err
	}
	if schedule, err := client.GetSchedule(pdScheduleId, pagerduty.GetScheduleOptions{
		Since: since.Format(time.RFC3339),
		Until: until.Format(time.RFC3339),
	}); err != nil {
		return []pagerduty.RenderedScheduleEntry{}, err
	} else {
		var entries []pagerduty.RenderedScheduleEntry
		for _, renderedScheduleEntry := range schedule.FinalSchedule.RenderedScheduleEntries {
			entries = append(entries, renderedScheduleEntry)
		}
		return entries, nil
	}
}

func makePdf(entries []pagerduty.RenderedScheduleEntry) ([]string, error) {
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

		pdf.Image(templateName, 0, 0, 210, 0, false, "", 0, "")

		pdf.SetXY(57, 56)
		pdf.Write(lineHt, name)

		until, err := tparse.ParseNow(time.RFC3339, tparseUntilQuery)
		if err != nil {
			return nil, err
		}
		pdf.SetXY(57, 63)
		pdf.Write(lineHt, until.Format("Monday, 02 Jan 2006"))

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

		fileStr := periods[0].from.Format("2006-01-02") + " until " + periods[len(periods)-1].to.Format("2006-01-02") + " " + name + " support oncall claim.pdf"
		results = append(results, fileStr)

		if err := pdf.OutputFileAndClose(fileStr); err != nil {
			fmt.Println("Unable to generate "+fileStr+" because:", err)
		} else {
			fmt.Println("File " + fileStr + " generated.")
		}
	}
	return results, nil
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Println("Error", err)
		os.Exit(-1)
	}
}
