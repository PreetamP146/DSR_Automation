package worker

import (
	"context"
	"dsr-automation/internal/services"
	"log"
	"time"
)

func StartCommitSyncWorker(ctx context.Context, syncService services.CommitSyncService, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	run := func() {
		result, err := syncService.SyncAll(ctx)
		if err != nil {
			log.Printf("commit sync worker: %v", err)
			return
		}
		log.Printf("commit sync worker: synced %d project(s), added %d commit(s)", result.ProjectsSynced, result.CommitsAdded)
	}

	run()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("commit sync worker stopped")
			return
		case <-ticker.C:
			run()
		}
	}
}
