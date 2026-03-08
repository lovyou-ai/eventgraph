package sqlitestore_test

import (
	"testing"

	"github.com/lovyou-ai/eventgraph/go/pkg/store"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/sqlitestore"
	"github.com/lovyou-ai/eventgraph/go/pkg/store/storetest"
)

func TestSQLiteConformance(t *testing.T) {
	storetest.RunConformanceSuite(t, func() store.Store {
		s, err := sqlitestore.New(":memory:")
		if err != nil {
			t.Fatalf("sqlitestore.New: %v", err)
		}
		return s
	})
}
