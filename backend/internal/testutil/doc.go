// Package testutil provides shared utilities for integration testing.
//
// This package consolidates common test infrastructure:
//   - Embedded Postgres setup (StartTestPostgres)
//   - Miniredis setup (StartTestRedis, StartMiniRedis)
//   - Test configuration (NewTestConfig)
//   - Test fixtures (Fixtures)
//
// Example usage:
//
//	func TestMyIntegration(t *testing.T) {
//	    ctx := context.Background()
//
//	    // Start test infrastructure
//	    pgStop, pgURL := testutil.StartTestPostgres(t)
//	    defer pgStop()
//
//	    redis := testutil.StartMiniRedis(t)
//
//	    // Apply schema
//	    if err := testutil.ApplySchema(ctx, pgURL); err != nil {
//	        t.Fatalf("apply schema: %v", err)
//	    }
//
//	    // Create config
//	    cfg := testutil.NewTestConfig(testutil.TestConfigOptions{
//	        PostgresURL: pgURL,
//	        RedisURL:    redis.URL(),
//	        FilesDir:    t.TempDir(),
//	    })
//
//	    // Run your tests...
//	}
package testutil
