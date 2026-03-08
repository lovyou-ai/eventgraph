package mysqlstore_test

import (
	"context"
	"os"
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/mysqlstore"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/storetest"
)

func TestMySQLConformance(t *testing.T) {
	dsn := os.Getenv("EVENTGRAPH_MYSQL_URL")
	if dsn == "" {
		t.Skip("EVENTGRAPH_MYSQL_URL not set; skipping MySQLStore conformance tests")
	}

	ctx := context.Background()

	storetest.RunConformanceSuite(t, func() store.Store {
		s, err := mysqlstore.New(dsn)
		if err != nil {
			t.Fatalf("mysqlstore.New: %v", err)
		}
		if err := s.Truncate(ctx); err != nil {
			t.Fatalf("Truncate: %v", err)
		}
		return s
	})
}
