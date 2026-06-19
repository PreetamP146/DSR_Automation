package worker

import (
	"context"
	"dsr-automation/internal/services"
	"log"

	"github.com/robfig/cron/v3"
)

func StartActivitySyncCron(ctx context.Context, trackerService services.ActivityTrackerService, cronSpec string) {
	if cronSpec == "" {
		cronSpec = "*/15 * * * *"
	}

	run := func() {
		result, err := trackerService.SyncAll(ctx)
		if err != nil {
			log.Printf("activity sync cron: %v", err)
			return
		}
		log.Printf(
			"activity sync cron: local repos=%d commits=%d planning integrations=%d activities=%d",
			result.LocalReposSynced,
			result.LocalCommitsAdded,
			result.PlanningIntegrationsSynced,
			result.PlanningActivitiesAdded,
		)
	}

	run()

	c := cron.New()
	if _, err := c.AddFunc(cronSpec, run); err != nil {
		log.Printf("activity sync cron: invalid schedule %q, using */15 * * * *: %v", cronSpec, err)
		if _, err := c.AddFunc("*/15 * * * *", run); err != nil {
			log.Printf("activity sync cron: failed to start: %v", err)
			return
		}
	}

	c.Start()
	log.Printf("activity sync cron started with schedule %s", cronSpec)

	<-ctx.Done()
	ctx2 := c.Stop()
	<-ctx2.Done()
	log.Println("activity sync cron stopped")
}
