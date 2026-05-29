package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type scheduledRunnerPlanRepoStub struct {
	claimDueCalls    int32
	release          chan struct{}
	claimDueFn       func(ctx context.Context, now time.Time, leaseUntil time.Time, limit int) ([]*ScheduledTestPlan, error)
	updateAfterRunFn func(ctx context.Context, id int64, claimedUntil time.Time, lastRunAt time.Time, nextRunAt time.Time) (bool, error)
}

func (s *scheduledRunnerPlanRepoStub) Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	panic("unexpected Create call")
}

func (s *scheduledRunnerPlanRepoStub) GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	panic("unexpected GetByID call")
}

func (s *scheduledRunnerPlanRepoStub) ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	panic("unexpected ListByAccountID call")
}

func (s *scheduledRunnerPlanRepoStub) ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error) {
	panic("unexpected ListDue call")
}

func (s *scheduledRunnerPlanRepoStub) ClaimDue(ctx context.Context, now time.Time, leaseUntil time.Time, limit int) ([]*ScheduledTestPlan, error) {
	atomic.AddInt32(&s.claimDueCalls, 1)
	if s.claimDueFn != nil {
		return s.claimDueFn(ctx, now, leaseUntil, limit)
	}
	if s.release != nil {
		<-s.release
	}
	return nil, nil
}

func (s *scheduledRunnerPlanRepoStub) Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	panic("unexpected Update call")
}

func (s *scheduledRunnerPlanRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *scheduledRunnerPlanRepoStub) UpdateAfterRun(ctx context.Context, id int64, claimedUntil time.Time, lastRunAt time.Time, nextRunAt time.Time) (bool, error) {
	if s.updateAfterRunFn != nil {
		return s.updateAfterRunFn(ctx, id, claimedUntil, lastRunAt, nextRunAt)
	}
	panic("unexpected UpdateAfterRun call")
}

type scheduledRunnerResultRepoStub struct {
	createFn func(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error)
	pruneFn  func(ctx context.Context, planID int64, keepCount int) error
}

func (s *scheduledRunnerResultRepoStub) Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	if s.createFn != nil {
		return s.createFn(ctx, result)
	}
	panic("unexpected Create call")
}

func (s *scheduledRunnerResultRepoStub) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	panic("unexpected ListByPlanID call")
}

func (s *scheduledRunnerResultRepoStub) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	if s.pruneFn != nil {
		return s.pruneFn(ctx, planID, keepCount)
	}
	panic("unexpected PruneOldResults call")
}

type scheduledRunnerAccountTesterStub struct {
	runFn func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

func (s *scheduledRunnerAccountTesterStub) RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	if s.runFn != nil {
		return s.runFn(ctx, accountID, modelID)
	}
	panic("unexpected RunTestBackground call")
}

func TestScheduledTestRunnerSkipsOverlappingRun(t *testing.T) {
	release := make(chan struct{})
	repo := &scheduledRunnerPlanRepoStub{release: release}
	runner := &ScheduledTestRunnerService{
		planRepo:   repo,
		tickDelay:  0,
		claimLease: scheduledTestClaimLease,
		claimLimit: 1,
		nowFunc:    time.Now,
		running:    atomic.Int32{},
		startOnce:  sync.Once{},
		stopOnce:   sync.Once{},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		runner.runScheduled()
	}()

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&repo.claimDueCalls) == 1
	}, time.Second, 10*time.Millisecond)

	runner.runScheduled()
	require.Equal(t, int32(1), atomic.LoadInt32(&repo.claimDueCalls))

	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("first scheduled run did not finish")
	}
}

func TestScheduledTestRunnerConcurrentRunnersDoNotExecuteSameClaimTwice(t *testing.T) {
	fixedNow := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	planTemplate := &ScheduledTestPlan{
		ID:             101,
		AccountID:      42,
		ModelID:        "gpt-4.1-mini",
		CronExpression: "* * * * *",
		Enabled:        true,
		MaxResults:     3,
	}

	var claimed atomic.Bool
	var runCalls atomic.Int32
	var resultCreateCalls atomic.Int32
	var resultPruneCalls atomic.Int32
	var updateCalls atomic.Int32

	repo := &scheduledRunnerPlanRepoStub{
		claimDueFn: func(ctx context.Context, now time.Time, leaseUntil time.Time, limit int) ([]*ScheduledTestPlan, error) {
			if claimed.CompareAndSwap(false, true) {
				plan := *planTemplate
				return []*ScheduledTestPlan{&plan}, nil
			}
			return nil, nil
		},
		updateAfterRunFn: func(ctx context.Context, id int64, claimedUntil time.Time, lastRunAt time.Time, nextRunAt time.Time) (bool, error) {
			updateCalls.Add(1)
			return true, nil
		},
	}
	resultRepo := &scheduledRunnerResultRepoStub{
		createFn: func(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
			resultCreateCalls.Add(1)
			return result, nil
		},
		pruneFn: func(ctx context.Context, planID int64, keepCount int) error {
			resultPruneCalls.Add(1)
			return nil
		},
	}
	tester := &scheduledRunnerAccountTesterStub{
		runFn: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			runCalls.Add(1)
			return &ScheduledTestResult{Status: "failed"}, nil
		},
	}
	scheduledSvc := NewScheduledTestService(repo, resultRepo)
	runner1 := &ScheduledTestRunnerService{
		planRepo:       repo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: tester,
		tickDelay:      0,
		claimLimit:     1,
		claimLease:     scheduledTestClaimLease,
		nowFunc:        func() time.Time { return fixedNow },
	}
	runner2 := &ScheduledTestRunnerService{
		planRepo:       repo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: tester,
		tickDelay:      0,
		claimLimit:     1,
		claimLease:     scheduledTestClaimLease,
		nowFunc:        func() time.Time { return fixedNow },
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, runner := range []*ScheduledTestRunnerService{runner1, runner2} {
		wg.Add(1)
		go func(r *ScheduledTestRunnerService) {
			defer wg.Done()
			<-start
			r.runScheduled()
		}(runner)
	}

	close(start)
	wg.Wait()

	require.Equal(t, int32(1), runCalls.Load())
	require.Equal(t, int32(1), resultCreateCalls.Load())
	require.Equal(t, int32(1), resultPruneCalls.Load())
	require.Equal(t, int32(1), updateCalls.Load())
}

func TestScheduledTestRunnerRunOnePlanAdvancesNextRunAfterExecution(t *testing.T) {
	fixedNow := time.Date(2026, 5, 28, 10, 20, 0, 0, time.UTC)
	plan := &ScheduledTestPlan{
		ID:             7,
		AccountID:      99,
		ModelID:        "gpt-4.1",
		CronExpression: "0 0 1 1 *",
		MaxResults:     4,
		NextRunAt:      ptrTime(fixedNow.Add(scheduledTestClaimLease)),
	}

	var (
		mu        sync.Mutex
		calls     []string
		lastRunAt time.Time
		nextRunAt time.Time
	)
	recordCall := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, name)
	}

	repo := &scheduledRunnerPlanRepoStub{
		updateAfterRunFn: func(ctx context.Context, id int64, claimedUntil time.Time, last time.Time, next time.Time) (bool, error) {
			recordCall("update")
			lastRunAt = last
			nextRunAt = next
			require.NotZero(t, claimedUntil)
			return true, nil
		},
	}
	resultRepo := &scheduledRunnerResultRepoStub{
		createFn: func(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
			recordCall("create")
			return result, nil
		},
		pruneFn: func(ctx context.Context, planID int64, keepCount int) error {
			recordCall("prune")
			return nil
		},
	}
	tester := &scheduledRunnerAccountTesterStub{
		runFn: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			recordCall("run")
			return &ScheduledTestResult{Status: "success"}, nil
		},
	}
	runner := &ScheduledTestRunnerService{
		planRepo:       repo,
		scheduledSvc:   NewScheduledTestService(repo, resultRepo),
		accountTestSvc: tester,
		nowFunc:        func() time.Time { return fixedNow },
	}

	runner.runOnePlan(context.Background(), plan)

	expectedNextRun, err := computeNextRun(plan.CronExpression, fixedNow)
	require.NoError(t, err)

	require.Equal(t, []string{"run", "create", "prune", "update"}, calls)
	require.Equal(t, fixedNow, lastRunAt)
	require.Equal(t, expectedNextRun, nextRunAt)
}
