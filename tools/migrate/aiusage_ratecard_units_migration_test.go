package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAIUsageRatecardUnitsMigrationAddsEntFields(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("migrations", "20260809000100_ai_usage_ratecard_units.up.sql"))
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile(filepath.Join("migrations", "20260809000100_ai_usage_ratecard_units.down.sql"))
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	upSQL := strings.ReplaceAll(strings.ToLower(string(up)), `"`, "")
	for _, column := range []string{"credits_per_unit", "unit_size"} {
		if !strings.Contains(upSQL, "add column if not exists "+column) {
			t.Fatalf("up migration must add %s idempotently", column)
		}
	}
	if !strings.Contains(upSQL, "default 1") {
		t.Fatal("up migration must default new unit columns to 1")
	}

	downSQL := strings.ReplaceAll(strings.ToLower(string(down)), `"`, "")
	for _, column := range []string{"credits_per_unit", "unit_size"} {
		if !strings.Contains(downSQL, "drop column if exists "+column) {
			t.Fatalf("down migration must drop %s idempotently", column)
		}
	}
}

func TestAIUsageRatecardPriceMigrationAllowsConfigSeed(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("migrations", "20260809000200_aiusage_ratecard_price_nullable.up.sql"))
	if err != nil {
		t.Fatalf("read price compatibility migration: %v", err)
	}

	upSQL := strings.ReplaceAll(strings.ToLower(string(up)), `"`, "")
	if !strings.Contains(upSQL, "alter column price_per_unit_cny drop not null") {
		t.Fatal("price compatibility migration must make the legacy price column nullable")
	}
}

func TestAIUsageRatecardPriceDefaultMigrationBackfillsLegacyRows(t *testing.T) {
	up, err := os.ReadFile(filepath.Join("migrations", "20260809000300_aiusage_ratecard_price_default.up.sql"))
	if err != nil {
		t.Fatalf("read price default migration: %v", err)
	}

	upSQL := strings.ReplaceAll(strings.ToLower(string(up)), `"`, "")
	if !strings.Contains(upSQL, "where price_per_unit_cny is null") {
		t.Fatal("price default migration must backfill existing null prices")
	}
	if !strings.Contains(upSQL, "alter column price_per_unit_cny set default 0") {
		t.Fatal("price default migration must default new config-seeded rows to zero")
	}
	if !strings.Contains(upSQL, "alter column price_per_unit_cny set not null") {
		t.Fatal("price default migration must prevent nil scan values after backfill")
	}
}
