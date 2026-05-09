package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const affiliateDistributionMonthlyResetCheckInterval = time.Hour

type AffiliateDistributionMonthlyResetService struct {
	affiliateService *AffiliateService
	now              func() time.Time
	stopCh           chan struct{}
	doneCh           chan struct{}
	once             sync.Once
}

func NewAffiliateDistributionMonthlyResetService(affiliateService *AffiliateService) *AffiliateDistributionMonthlyResetService {
	svc := &AffiliateDistributionMonthlyResetService{
		affiliateService: affiliateService,
		now:              time.Now,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}
	go svc.run()
	return svc
}

func (s *AffiliateDistributionMonthlyResetService) Stop() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		close(s.stopCh)
		<-s.doneCh
	})
}

func (s *AffiliateDistributionMonthlyResetService) run() {
	defer close(s.doneCh)
	s.archiveIfDue(context.Background())
	ticker := time.NewTicker(affiliateDistributionMonthlyResetCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.archiveIfDue(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *AffiliateDistributionMonthlyResetService) archiveIfDue(ctx context.Context) {
	if s == nil || s.affiliateService == nil {
		return
	}
	now := s.now().UTC()
	if now.Day() != 1 {
		return
	}
	archiveMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	count, err := s.affiliateService.ArchiveMonthlyRebateBalances(ctx, archiveMonth, nil, "system")
	if err != nil {
		logger.LegacyPrintf("service.affiliate_distribution", "monthly rebate archive failed: month=%s err=%v", archiveMonth.Format("2006-01"), err)
		return
	}
	if count > 0 {
		logger.LegacyPrintf("service.affiliate_distribution", "monthly rebate archive completed: month=%s archived=%d", archiveMonth.Format("2006-01"), count)
	}
}
