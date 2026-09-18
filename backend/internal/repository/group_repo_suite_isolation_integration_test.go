//go:build integration

package repository

import "github.com/Wei-Shaw/sub2api/internal/service"

// BeforeTest keeps seed-data growth from making tests that intentionally
// exercise a fixed baseline brittle. Production data and repository logic are
// untouched; this normalization only runs inside the integration-test
// transaction for the named test below.
func (s *GroupRepoSuite) BeforeTest(_, testName string) {
	if testName != "TestListActiveByPlatform" {
		return
	}

	_, err := s.tx.ExecContext(
		s.ctx,
		`UPDATE groups
SET status = $1
WHERE platform = $2
  AND status = $3
  AND id <> (
    SELECT id
    FROM groups
    WHERE platform = $2 AND status = $3
    ORDER BY id
    LIMIT 1
  )`,
		service.StatusDisabled,
		service.PlatformAnthropic,
		service.StatusActive,
	)
	s.Require().NoError(err, "normalize active anthropic seed groups")
}
