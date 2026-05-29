package schema

import (
	"testing"

	"entgo.io/ent/entc/load"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestTimeMixinBackedSchemas(t *testing.T) {
	spec, err := (&load.Config{Path: "."}).Load()
	require.NoError(t, err)

	schemas := map[string]*load.Schema{}
	for _, schema := range spec.Schemas {
		schemas[schema.Name] = schema
	}

	for _, schemaName := range []string{
		"Announcement",
		"PaymentOrder",
		"PaymentProviderInstance",
		"PromoCode",
		"SubscriptionPlan",
	} {
		schema := requireSchema(t, schemas, schemaName)
		requireMixedInField(t, schema, "created_at")
		requireMixedInField(t, schema, "updated_at")
	}
}

func TestUserEmailPartialUniqueIndexIsDeclaredInSchema(t *testing.T) {
	spec, err := (&load.Config{Path: "."}).Load()
	require.NoError(t, err)

	schemas := map[string]*load.Schema{}
	for _, schema := range spec.Schemas {
		schemas[schema.Name] = schema
	}

	userSchema := requireSchema(t, schemas, "User")
	emailIndex := requireSchemaIndex(t, userSchema, "email")
	require.True(t, emailIndex.Unique)
	require.Equal(t, "users_email_unique_active", emailIndex.StorageKey)
	require.Equal(t, "deleted_at IS NULL", requireIndexWhere(t, emailIndex))
}

func TestAccountAndPaymentOrderEnumsAreExplicit(t *testing.T) {
	spec, err := (&load.Config{Path: "."}).Load()
	require.NoError(t, err)

	schemas := map[string]*load.Schema{}
	for _, schema := range spec.Schemas {
		schemas[schema.Name] = schema
	}

	accountSchema := requireSchema(t, schemas, "Account")
	requireFieldEnumValues(t, accountSchema, "platform",
		domain.PlatformAnthropic,
		domain.PlatformOpenAI,
		domain.PlatformGemini,
		domain.PlatformAntigravity,
		domain.PlatformOpenAICompatible,
		domain.PlatformAnthropicCompatible,
	)
	requireFieldEnumValues(t, accountSchema, "type",
		domain.AccountTypeOAuth,
		domain.AccountTypeSetupToken,
		domain.AccountTypeAPIKey,
		domain.AccountTypeUpstream,
		domain.AccountTypeBedrock,
		domain.AccountTypeServiceAccount,
	)
	requireFieldEnumValues(t, accountSchema, "status",
		domain.StatusActive,
		domain.StatusDisabled,
		domain.StatusError,
	)

	paymentOrderSchema := requireSchema(t, schemas, "PaymentOrder")
	requireFieldEnumValues(t, paymentOrderSchema, "order_type",
		payment.OrderTypeBalance,
		payment.OrderTypeSubscription,
	)
	requireFieldEnumValues(t, paymentOrderSchema, "status",
		payment.OrderStatusPending,
		payment.OrderStatusPaid,
		payment.OrderStatusRecharging,
		payment.OrderStatusCompleted,
		payment.OrderStatusExpired,
		payment.OrderStatusCancelled,
		payment.OrderStatusFailed,
		payment.OrderStatusRefundRequested,
		payment.OrderStatusRefunding,
		payment.OrderStatusPartiallyRefunded,
		payment.OrderStatusRefunded,
		payment.OrderStatusRefundFailed,
	)
}

func requireMixedInField(t *testing.T, schema *load.Schema, fieldName string) {
	t.Helper()

	schemaField := requireSchemaField(t, schema, fieldName)
	require.NotNil(t, schemaField.Position, "field %s on schema %s should record its position", fieldName, schema.Name)
	require.True(t, schemaField.Position.MixedIn, "field %s on schema %s should come from a mixin", fieldName, schema.Name)
}

func requireSchemaIndex(t *testing.T, schema *load.Schema, fields ...string) *load.Index {
	t.Helper()

	for _, schemaIndex := range schema.Indexes {
		if len(schemaIndex.Fields) != len(fields) {
			continue
		}
		match := true
		for i := range fields {
			if schemaIndex.Fields[i] != fields[i] {
				match = false
				break
			}
		}
		if match {
			return schemaIndex
		}
	}

	require.Failf(t, "missing schema index", "schema %s should include index on %v", schema.Name, fields)
	return nil
}

func requireIndexWhere(t *testing.T, schemaIndex *load.Index) string {
	t.Helper()

	raw, ok := schemaIndex.Annotations["EntSQLIndexes"]
	require.True(t, ok, "index %v should include EntSQLIndexes annotation", schemaIndex.Fields)

	annotation, ok := raw.(map[string]any)
	require.True(t, ok, "EntSQLIndexes annotation should decode to a map")

	if where, ok := annotation["Where"].(string); ok {
		return where
	}
	if where, ok := annotation["where"].(string); ok {
		return where
	}

	require.Failf(t, "missing partial index predicate", "index %v should declare a WHERE predicate", schemaIndex.Fields)
	return ""
}

func requireFieldEnumValues(t *testing.T, schema *load.Schema, fieldName string, expected ...string) {
	t.Helper()

	schemaField := requireSchemaField(t, schema, fieldName)
	actual := make([]string, 0, len(schemaField.Enums))
	for _, enumValue := range schemaField.Enums {
		actual = append(actual, enumValue.V)
	}
	require.Equal(t, expected, actual)
}
