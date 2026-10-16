//go:build testing

package hub

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"
	pbcron "github.com/pocketbase/pocketbase/tools/cron"
	"github.com/stretchr/testify/require"
)

func TestHistoryCronSchedulesAroundStartupBoundary(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	h := NewHub(app)
	require.NoError(t, h.registerCronJobs(nil))
	jobs := h.Cron().Jobs()

	before := time.Date(2026, time.January, 15, 8, 7, 0, 0, time.UTC)
	afterCleanupTick := time.Date(2026, time.January, 15, 8, 9, 0, 0, time.UTC)
	cleanupDue := time.Date(2026, time.January, 15, 9, 8, 0, 0, time.UTC)
	rollupDue := time.Date(2026, time.January, 15, 8, 10, 0, 0, time.UTC)
	foundDelete := false
	foundRollup := false
	for _, job := range jobs {
		if job.Id() != "delete old records" && job.Id() != "create longer records" {
			continue // PocketBase registers its own maintenance jobs on bootstrap.
		}
		schedule, err := pbcron.NewSchedule(job.Expression())
		require.NoError(t, err)
		switch job.Id() {
		case "delete old records":
			require.Equal(t, "8 * * * *", job.Expression())
			require.False(t, schedule.IsDue(pbcron.NewMoment(before)))
			require.False(t, schedule.IsDue(pbcron.NewMoment(afterCleanupTick)), "a startup just after minute 8 must not trigger a missed run")
			require.True(t, schedule.IsDue(pbcron.NewMoment(cleanupDue)), "the next hourly cleanup remains scheduled")
			foundDelete = true
		case "create longer records":
			require.Equal(t, "*/10 * * * *", job.Expression())
			require.False(t, schedule.IsDue(pbcron.NewMoment(afterCleanupTick)))
			require.True(t, schedule.IsDue(pbcron.NewMoment(rollupDue)), "rollup should run at the next ten-minute boundary")
			foundRollup = true
		default:
			t.Fatalf("unexpected registered Hub history job: %s (%s)", job.Id(), job.Expression())
		}
	}
	require.True(t, foundDelete, "the hourly cleanup job must be registered")
	require.True(t, foundRollup, "the ten-minute rollup job must be registered")
}
