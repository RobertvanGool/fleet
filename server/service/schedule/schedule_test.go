package schedule

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func nopStatsHandler(interface{}, error) {}

type nopLocker struct{}

func (nopLocker) Lock(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}

func (nopLocker) Unlock(context.Context, string, string) error {
	return nil
}

func TestNewSchedule(t *testing.T) {
	sched, err := New(context.Background(), "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)

	sched.AddJob("test_job", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, nil
	}, nopStatsHandler)

	runCount := 0
	failCheck := time.After(1 * time.Second)

TEST:
	for {
		select {
		case <-runCheck:
			runCount++
			if runCount > 2 {
				break TEST
			}
		case <-failCheck:
			assert.Greater(t, runCount, 2)
			t.FailNow()
		}
	}
}

func TestStatsHandler(t *testing.T) {
	sched, err := New(context.Background(), "test_stats_handler", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)

	sched.AddJob("test_job", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return "stats foo", fmt.Errorf("error foo")
	}, func(stats interface{}, err error) {
		require.Equal(t, "stats foo", stats)
		require.Equal(t, fmt.Errorf("error foo"), err)
	})

	runCount := 0
	failCheck := time.After(1 * time.Second)

TEST:
	for {
		select {
		case <-runCheck:
			runCount++
			if runCount > 2 {
				break TEST
			}
		case <-failCheck:
			assert.Greater(t, runCount, 2)
			t.FailNow()
		}
	}
}

type testLocker struct {
	count *uint
}

func (l testLocker) Lock(context.Context, string, string, time.Duration) (bool, error) {
	*l.count++
	return true, nil
}

func (testLocker) Unlock(context.Context, string, string) error {
	return nil
}

func TestScheduleLocker(t *testing.T) {
	locker := testLocker{count: ptr.Uint(0)}

	sched, err := New(context.Background(), "test_schedule_locker", "test_instance", 10*time.Millisecond, locker, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)
	failCheck := time.After(1 * time.Second)

	sched.AddJob("test_job", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, nil
	}, nopStatsHandler)

TEST:
	for {
		select {
		case <-runCheck:
			sched.muChecks.Lock()
			if *locker.count > 2 {
				break TEST
			}
			sched.muChecks.Unlock()
		case <-failCheck:
			assert.Greater(t, *locker.count, uint(2))
			t.FailNow()
		}
	}
}

type testStats struct {
	mu     sync.Mutex
	stats  map[string][]interface{}
	errors map[string][]error
}

// TODO: Ask Tomas about how to structure mutex for stats?
func statsHandlerFunc(jobName string, ts *testStats) func(interface{}, error) {
	return func(stats interface{}, err error) {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		if err != nil {
			ts.errors[jobName] = append(ts.errors[jobName], err)
		}
		ts.stats[jobName] = append(ts.stats[jobName], stats)
	}
}

func TestMultipleSchedules(t *testing.T) {
	testStats := &testStats{
		stats:  make(map[string][]interface{}),
		errors: make(map[string][]error),
	}

	sched1, err := New(context.Background(), "test_schedule_1", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)

	sched1.AddJob("test_job_1", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return "stats_job_1", nil
	}, statsHandlerFunc("test_job_1", testStats))

	sched2, err := New(context.Background(), "test_schedule_2", "test_instance", 100*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	sched2.AddJob("test_job_2", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return "stats_job_2", nil
	}, statsHandlerFunc("test_job_2", testStats))

	sched3, err := New(context.Background(), "test_schedule_3", "test_instance", 100*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	sched3.AddJob("test_job_3", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, fmt.Errorf("error_job_3")
	}, statsHandlerFunc("test_job_3", testStats))

	failCheck := time.After(1 * time.Second)

TEST:
	for {
		select {
		case <-runCheck:
			testStats.mu.Lock()
			if (len(testStats.stats["test_job_1"]) > 2) && (len(testStats.stats["test_job_2"]) > 2) && (len(testStats.stats["test_job_3"]) > 2) {
				break TEST
			}
			testStats.mu.Unlock()
		case <-failCheck:
			testStats.mu.Lock()
			assert.Greater(t, len(testStats.stats["test_job_1"]), 2)
			assert.Greater(t, len(testStats.stats["test_job_2"]), 2)
			assert.Greater(t, len(testStats.errors["test_job_3"]), 2)
			testStats.mu.Unlock()
			t.FailNow()
		}
	}
}

func TestPreflightCheck(t *testing.T) {
	preflightFailed := make(chan bool)

	sched, err := New(context.Background(), "test_schedule_1", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	sched.SetPreflightCheck(func() bool {
		preflightFailed <- true
		return false
	})

	sched.AddJob("test_job_1", func(ctx context.Context) (interface{}, error) {
		t.FailNow() // preflight should fail so the job should never run
		return nil, nil
	}, nopStatsHandler)

	failCheck := time.After(30 * time.Millisecond)

TEST:
	for {
		select {
		case <-preflightFailed:
			break TEST
		case <-failCheck:
			t.FailNow()
		}
	}
}

func TestConfigCheck(t *testing.T) {
	runCheck := make(chan bool)

	sched, err := New(context.Background(), "test_schedule_1", "test_instance", 200*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	sched.AddJob("test_job_1", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, nil
	}, nopStatsHandler)

	sched.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		newInterval := 20 * time.Millisecond
		return &newInterval, nil
	})

	failCheck := time.After(300 * time.Millisecond)

TEST:
	for {
		select {
		case <-runCheck:
			sched.muChecks.Lock()
			if sched.interval == 20*time.Millisecond {
				break TEST
			}
			sched.muChecks.Unlock()
		case <-failCheck:
			require.Equal(t, 20*time.Millisecond, sched.interval)
			// fmt.Println(sched.interval)
			// if sched.interval == 20*time.Millisecond {
			// 	break TEST
			// }
			t.FailNow()
		}
	}
}

func TestTickerRaces(t *testing.T) {
	schedTicker := time.NewTicker(2 * time.Hour)

	go func() {
		for {
			schedTicker.Reset(1 * time.Hour)
		}
	}()
	go func() {
		for {
			schedTicker.Reset(3 * time.Hour)
		}
	}()
	time.Sleep(10 * time.Second)
}
