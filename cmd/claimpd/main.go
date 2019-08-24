package main

import (
	"errors"
	"fmt"
	"github.com/faizhasim/gdp-pagerduty-claim/pkg/pdfgenerator"
	"github.com/karrick/tparse"
	"github.com/spf13/cobra"
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

var (
	authtoken        string
	scheduleName     string
	tparseSinceQuery string
	tparseUntilQuery string
)

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
			claimFormImageFilePath, err := pdfgenerator.DownloadPdfTemplate()
			if err != nil {
				return err
			}

			since, err := tparse.ParseNow(time.RFC3339, tparseSinceQuery)
			if err != nil {
				return err
			}

			until, err := tparse.ParseNow(time.RFC3339, tparseUntilQuery)
			if err != nil {
				return err
			}

			entries, _ := pdfgenerator.FetchScheduleEntries(since, until, authtoken, scheduleName)

			dirPath := "./"
			_, err = pdfgenerator.MakePdfToDir(claimFormImageFilePath, dirPath, entries, time.Now())
			return err
		},
	}

	cmd.Flags().StringVarP(&authtoken, "pd-api-key", "p", "", "PagerDuty v2 API Key")
	cmd.Flags().StringVarP(&scheduleName, "schedule-name", "s", "", "PagerDuty v2 Schedule Name for Query")
	cmd.Flags().StringVarP(&tparseSinceQuery, "since", "", "now-1mo", "Generate report since when? Example: now-2w or now-3mo")
	cmd.Flags().StringVarP(&tparseUntilQuery, "until", "", "now", "Generate report until when? Example: now-2w or now-3mo ")

	return cmd
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Println("Error", err)
		os.Exit(-1)
	}
}
