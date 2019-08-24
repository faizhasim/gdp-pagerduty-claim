package pdfgenerator

import (
	"github.com/PagerDuty/go-pagerduty"
	"time"
)

func FetchScheduleEntries(since, until time.Time, authtoken, scheduleQuery string) ([]pagerduty.RenderedScheduleEntry, error) {
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
