package service

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/collection"
)

func TestProvideTimingWheelService_ReturnsError(t *testing.T) {
	original := newTimingWheel
	t.Cleanup(func() { newTimingWheel = original })

	newTimingWheel = func(_ time.Duration, _ int, _ collection.Execute) (*collection.TimingWheel, error) {
		return nil, errors.New("boom")
	}

	svc, err := ProvideTimingWheelService()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if svc != nil {
		t.Fatal("expected nil service on constructor error")
	}
}

func TestProvideTimingWheelService_Success(t *testing.T) {
	svc, err := ProvideTimingWheelService()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if svc == nil {
		t.Fatal("expected service, got nil")
	}
	svc.Stop()
}

type stubAffiliateDistributionUsageSettlementService struct{}

func (*stubAffiliateDistributionUsageSettlementService) SettleUsage(context.Context, AffiliateDistributionUsageSettlementCommand) (bool, error) {
	return false, nil
}

func TestProvideGatewayService_WiresAffiliateDistributionSettlementService(t *testing.T) {
	t.Parallel()

	settlementService := &stubAffiliateDistributionUsageSettlementService{}

	svc := ProvideGatewayService(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		settlementService,
	)

	if svc == nil {
		t.Fatal("expected GatewayService, got nil")
	}
	if svc.affiliateDistributionSettlementService != settlementService {
		t.Fatal("expected GatewayService to keep the injected affiliate distribution settlement service")
	}
}

func TestProvideOpenAIGatewayService_WiresAffiliateDistributionSettlementService(t *testing.T) {
	t.Parallel()

	settlementService := &stubAffiliateDistributionUsageSettlementService{}

	svc := ProvideOpenAIGatewayService(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		settlementService,
	)

	if svc == nil {
		t.Fatal("expected OpenAIGatewayService, got nil")
	}
	if svc.affiliateDistributionSettlementService != settlementService {
		t.Fatal("expected OpenAIGatewayService to keep the injected affiliate distribution settlement service")
	}
}

func TestProviderSet_ExposesAffiliateDistributionSettlementGatewayPath(t *testing.T) {
	t.Parallel()

	providers := providerSetEntries(t)

	for _, name := range []string{
		"ProvideGatewayService",
		"ProvideOpenAIGatewayService",
		"ProvideAffiliateDistributionSettlementService",
	} {
		if !providers[name] {
			t.Fatalf("expected ProviderSet to include %s", name)
		}
	}
}

func TestProvideAffiliateDistributionSettlementProviders_ReturnErrorsWithoutRepository(t *testing.T) {
	t.Parallel()

	_, err := ProvideAffiliateDistributionUsageSettlementStore(nil)
	require.Error(t, err)

	_, err = ProvideAffiliateDistributionUsageSettlementProcessor(nil)
	require.Error(t, err)

	_, err = ProvideAffiliateDistributionSettlementService(nil, nil)
	require.Error(t, err)
}

func providerSetEntries(t *testing.T) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "wire.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse wire.go: %v", err)
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) == 0 || valueSpec.Names[0].Name != "ProviderSet" || len(valueSpec.Values) == 0 {
				continue
			}

			callExpr, ok := valueSpec.Values[0].(*ast.CallExpr)
			if !ok {
				t.Fatal("ProviderSet is not initialized by a call expression")
			}
			selector, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				t.Fatal("ProviderSet is not initialized with wire.NewSet")
			}
			pkgIdent, ok := selector.X.(*ast.Ident)
			if !ok || pkgIdent.Name != "wire" || selector.Sel.Name != "NewSet" {
				t.Fatal("ProviderSet is not initialized with wire.NewSet")
			}

			providers := make(map[string]bool, len(callExpr.Args))
			for _, arg := range callExpr.Args {
				if ident, ok := arg.(*ast.Ident); ok {
					providers[ident.Name] = true
				}
			}
			return providers
		}
	}

	t.Fatal("ProviderSet not found in wire.go")
	return nil
}
