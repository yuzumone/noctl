package export

import (
	"context"
	"testing"

	"github.com/jomei/notionapi"
)

type databaseSource struct{ *fakeSource }

func (s databaseSource) QueryDatabaseAll(context.Context, string) ([]notionapi.Page, error) {
	return []notionapi.Page{{ID: "root"}, {ID: "second"}}, nil
}
func TestDatabase(t *testing.T) {
	s := databaseSource{testSource("Hello")}
	s.properties["second"] = map[string]any{"slug": "second"}
	docs, err := Database(context.Background(), s, "db", "")
	if err != nil || len(docs) != 2 {
		t.Fatalf("documents: %v, error: %v", docs, err)
	}
	s.properties["second"]["slug"] = "EXAMPLE"
	if _, err := Database(context.Background(), s, "db", ""); err == nil {
		t.Fatal("accepted duplicate filename")
	}
	s.properties["second"]["slug"] = "../invalid"
	if _, err := Database(context.Background(), s, "db", ""); err == nil {
		t.Fatal("accepted invalid slug")
	}
}
