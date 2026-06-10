package mediafiles

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/getvelora/velora/apps/server/internal/migrations"
	_ "modernc.org/sqlite"
)

func TestRepositoryReconcileTracksAddedUnchangedUpdatedMissingAndRestored(t *testing.T) {
	db := openRepositoryDB(t)
	repo := NewRepository(db, "sqlite")
	ctx := context.Background()
	firstScan := time.Date(2026, time.June, 1, 10, 0, 0, 0, time.UTC)
	modified := time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)

	first, err := repo.Reconcile(ctx, 1, []DiscoveredFile{
		{Path: "Movie.mkv", Size: 100, ModifiedAt: modified},
		{Path: "shows/Episode.MP4", Size: 200, ModifiedAt: modified},
	}, firstScan)
	if err != nil {
		t.Fatalf("first Reconcile: %v", err)
	}
	assertSummary(t, first, ReconcileSummary{Discovered: 2, Added: 2})

	secondScan := firstScan.Add(time.Hour)
	second, err := repo.Reconcile(ctx, 1, []DiscoveredFile{
		{Path: "Movie.mkv", Size: 101, ModifiedAt: modified.Add(time.Minute)},
		{Path: "new.webm", Size: 300, ModifiedAt: modified},
	}, secondScan)
	if err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	assertSummary(t, second, ReconcileSummary{
		Discovered:    2,
		Added:         1,
		Updated:       1,
		MarkedMissing: 1,
	})

	thirdScan := secondScan.Add(time.Hour)
	third, err := repo.Reconcile(ctx, 1, []DiscoveredFile{
		{Path: "Movie.mkv", Size: 101, ModifiedAt: modified.Add(time.Minute)},
		{Path: "new.webm", Size: 300, ModifiedAt: modified},
		{Path: "shows/Episode.MP4", Size: 200, ModifiedAt: modified},
	}, thirdScan)
	if err != nil {
		t.Fatalf("third Reconcile: %v", err)
	}
	assertSummary(t, third, ReconcileSummary{
		Discovered: 3,
		Unchanged:  2,
		Restored:   1,
	})

	files, err := repo.List(ctx, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0].Path != "Movie.mkv" || files[1].Path != "new.webm" ||
		files[2].Path != "shows/Episode.MP4" {
		t.Fatalf("unexpected order: %+v", files)
	}
	for _, file := range files {
		if file.Status != StatusAvailable || file.MissingAt != nil {
			t.Fatalf("expected available restored file, got %+v", file)
		}
		if !file.LastSeenAt.Equal(thirdScan) {
			t.Fatalf("expected last seen %s, got %s", thirdScan, file.LastSeenAt)
		}
	}
}

func TestRepositoryReconcileIsIsolatedByLibrary(t *testing.T) {
	db := openRepositoryDB(t)
	repo := NewRepository(db, "sqlite")
	now := time.Date(2026, time.June, 1, 10, 0, 0, 0, time.UTC)

	if _, err := repo.Reconcile(context.Background(), 1, []DiscoveredFile{
		{Path: "shared.mkv", Size: 100, ModifiedAt: now},
	}, now); err != nil {
		t.Fatalf("reconcile library 1: %v", err)
	}
	if _, err := repo.Reconcile(context.Background(), 2, []DiscoveredFile{
		{Path: "shared.mkv", Size: 200, ModifiedAt: now},
	}, now); err != nil {
		t.Fatalf("reconcile library 2: %v", err)
	}

	files, err := repo.List(context.Background(), 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 1 || files[0].Size != 200 {
		t.Fatalf("unexpected library 2 files: %+v", files)
	}
}

func TestRepositoryReconcileRollsBackOnFailure(t *testing.T) {
	db := openRepositoryDB(t)
	repo := NewRepository(db, "sqlite")
	now := time.Date(2026, time.June, 1, 10, 0, 0, 0, time.UTC)

	_, err := repo.Reconcile(context.Background(), 1, []DiscoveredFile{
		{Path: "duplicate.mkv", Size: 100, ModifiedAt: now},
		{Path: "duplicate.mkv", Size: 100, ModifiedAt: now},
	}, now)
	if err == nil {
		t.Fatal("expected duplicate snapshot to fail")
	}

	files, listErr := repo.List(context.Background(), 1)
	if listErr != nil {
		t.Fatalf("List: %v", listErr)
	}
	if len(files) != 0 {
		t.Fatalf("expected rollback to leave no files, got %+v", files)
	}
}

func TestRepositoryReconcilePersistsAndReplacesInspection(t *testing.T) {
	db := openRepositoryDB(t)
	repo := NewRepository(db, "sqlite")
	ctx := context.Background()
	firstScan := time.Date(2026, time.June, 1, 10, 0, 0, 0, time.UTC)
	modified := time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)

	first, err := repo.Reconcile(ctx, 1, []DiscoveredFile{{
		Path:       "Movie.mkv",
		Size:       100,
		ModifiedAt: modified,
		Inspection: InspectionResult{
			Attempted: true,
			Result: ProbeResult{
				Format: Format{
					Name:       "matroska,webm",
					LongName:   "Matroska / WebM",
					DurationMS: 60_000,
					BitRate:    8_000_000,
				},
				Streams: []Stream{{
					Index:         0,
					Type:          StreamTypeVideo,
					CodecName:     "hevc",
					Width:         3840,
					Height:        2160,
					ColorTransfer: "smpte2084",
				}},
			},
		},
	}}, firstScan)
	if err != nil {
		t.Fatalf("first Reconcile: %v", err)
	}
	assertSummary(t, first, ReconcileSummary{Discovered: 1, Added: 1, Probed: 1})

	files, err := repo.List(ctx, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 1 || files[0].Inspection.Status != InspectionStatusReady ||
		files[0].Inspection.Format == nil ||
		files[0].Inspection.Format.DurationMS != 60_000 ||
		len(files[0].Inspection.Streams) != 1 ||
		files[0].Inspection.Streams[0].CodecName != "hevc" {
		t.Fatalf("unexpected first inspection: %+v", files)
	}

	secondScan := firstScan.Add(time.Hour)
	second, err := repo.Reconcile(ctx, 1, []DiscoveredFile{{
		Path:       "Movie.mkv",
		Size:       101,
		ModifiedAt: modified.Add(time.Minute),
		Inspection: InspectionResult{
			Attempted: true,
			Result: ProbeResult{
				Format: Format{Name: "matroska,webm", DurationMS: 61_000},
				Streams: []Stream{{
					Index:     1,
					Type:      StreamTypeAudio,
					CodecName: "aac",
				}},
			},
		},
	}}, secondScan)
	if err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	assertSummary(t, second, ReconcileSummary{Discovered: 1, Updated: 1, Probed: 1})

	files, err = repo.List(ctx, 1)
	if err != nil {
		t.Fatalf("List after replacement: %v", err)
	}
	if len(files[0].Inspection.Streams) != 1 ||
		files[0].Inspection.Streams[0].CodecName != "aac" {
		t.Fatalf("expected replacement stream, got %+v", files[0].Inspection.Streams)
	}
}

func TestRepositoryFailedReinspectionClearsStaleMetadata(t *testing.T) {
	db := openRepositoryDB(t)
	repo := NewRepository(db, "sqlite")
	ctx := context.Background()
	firstScan := time.Date(2026, time.June, 1, 10, 0, 0, 0, time.UTC)
	modified := time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)

	_, err := repo.Reconcile(ctx, 1, []DiscoveredFile{{
		Path:       "Movie.mkv",
		Size:       100,
		ModifiedAt: modified,
		Inspection: InspectionResult{
			Attempted: true,
			Result: ProbeResult{
				Format:  Format{Name: "matroska,webm"},
				Streams: []Stream{{Index: 0, Type: StreamTypeVideo, CodecName: "hevc"}},
			},
		},
	}}, firstScan)
	if err != nil {
		t.Fatalf("first Reconcile: %v", err)
	}

	message := "invalid media"
	second, err := repo.Reconcile(ctx, 1, []DiscoveredFile{{
		Path:       "Movie.mkv",
		Size:       101,
		ModifiedAt: modified.Add(time.Minute),
		Inspection: InspectionResult{
			Attempted: true,
			Error:     message,
		},
	}}, firstScan.Add(time.Hour))
	if err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	assertSummary(t, second, ReconcileSummary{
		Discovered:  1,
		Updated:     1,
		Probed:      1,
		ProbeFailed: 1,
	})

	files, err := repo.List(ctx, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	inspection := files[0].Inspection
	if inspection.Status != InspectionStatusError || inspection.Error == nil ||
		*inspection.Error != message || inspection.Format != nil ||
		len(inspection.Streams) != 0 {
		t.Fatalf("unexpected failed inspection: %+v", inspection)
	}
}

func assertSummary(t *testing.T, got, want ReconcileSummary) {
	t.Helper()
	if got != want {
		t.Fatalf("summary mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func openRepositoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrations.Run(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	for _, library := range []struct {
		id   int
		name string
		path string
	}{
		{id: 1, name: "Movies", path: "/media/movies"},
		{id: 2, name: "TV", path: "/media/tv"},
	} {
		if _, err := db.Exec(`
			INSERT INTO libraries (id, name, path, created_at, updated_at)
			VALUES (?, ?, ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
		`, library.id, library.name, library.path); err != nil {
			t.Fatalf("insert library %d: %v", library.id, err)
		}
	}
	return db
}
