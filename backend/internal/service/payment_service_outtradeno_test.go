package service

import (
	"regexp"
	"testing"
)

func TestGenerateOutTradeNoFormat(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`^sub2_[0-9]{8}_[0-9a-f]{32}$`)
	outTradeNo := generateOutTradeNo()
	if !pattern.MatchString(outTradeNo) {
		t.Fatalf("out_trade_no format mismatch: %s", outTradeNo)
	}
}

func TestGenerateOutTradeNoUniqueness(t *testing.T) {
	t.Parallel()

	const sampleSize = 2000
	seen := make(map[string]struct{}, sampleSize)
	for i := 0; i < sampleSize; i++ {
		outTradeNo := generateOutTradeNo()
		if _, exists := seen[outTradeNo]; exists {
			t.Fatalf("duplicate out_trade_no generated: %s", outTradeNo)
		}
		seen[outTradeNo] = struct{}{}
	}
}
