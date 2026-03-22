package integration

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/domain"
)

const (
	pgUser            = "specto"
	pgPass            = "specto"
	pgDB              = "specto_test"
	aliceEmail        = "alice@example.com"
	bobEmail          = "bob@example.com"
	runIntegrationEnv = "SPECTO_RUN_INTEGRATION"
)

func requireIntegrationEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv(runIntegrationEnv) == "1" {
		return
	}
	t.Skipf("set %s=1 to run integration tests", runIntegrationEnv)
}

// startPostgres spins up an ephemeral PostgreSQL container and returns
// the connected *sql.DB. The container is terminated when the test ends.
func startPostgres(t *testing.T) *sql.DB {
	t.Helper()
	requireIntegrationEnabled(t)
	ctx := context.Background()

	// Resolve the project root so we can mount migration + fixture files.
	root := projectRoot(t)
	initDir := filepath.Join(root, "internal", "database", "migrations")
	fixtureFile := filepath.Join(root, "internal", "database", "fixtures", "fixtures.sql")

	ctr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(pgDB),
		postgres.WithUsername(pgUser),
		postgres.WithPassword(pgPass),
		postgres.WithInitScripts(
			filepath.Join(initDir, "001_init.up.sql"),
			fixtureFile,
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := database.OpenPostgres(dsn, logger)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// projectRoot walks up from the CWD until it finds go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// ---------- User Repository Tests ----------

func TestPgUserRepoCreateAndGetByID(t *testing.T) {
	db := startPostgres(t)
	repo := database.NewPgUserRepo(db)

	user := &domain.User{
		Email:    "integration@example.com",
		Name:     "Integration",
		Password: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012",
	}

	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected user ID to be assigned")
	}

	got, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if got.Email != user.Email {
		t.Fatalf("expected email %q, got %q", user.Email, got.Email)
	}
	if got.Name != "Integration" {
		t.Fatalf("expected name %q, got %q", "Integration", got.Name)
	}
}

func TestPgUserRepoGetByEmailFixtures(t *testing.T) {
	db := startPostgres(t)
	repo := database.NewPgUserRepo(db)

	// fixtures.sql inserts alice@example.com.
	alice, err := repo.GetByEmail(context.Background(), aliceEmail)
	if err != nil {
		t.Fatalf("get alice: %v", err)
	}
	if alice.Name != "Alice" {
		t.Fatalf("expected Alice, got %q", alice.Name)
	}
}

func TestPgUserRepoUpdate(t *testing.T) {
	db := startPostgres(t)
	repo := database.NewPgUserRepo(db)

	alice, _ := repo.GetByEmail(context.Background(), aliceEmail)
	alice.Name = "Alice Updated"
	if err := repo.Update(context.Background(), alice); err != nil {
		t.Fatalf("update user: %v", err)
	}

	got, _ := repo.GetByID(context.Background(), alice.ID)
	if got.Name != "Alice Updated" {
		t.Fatalf("expected 'Alice Updated', got %q", got.Name)
	}
}

func TestPgUserRepoDelete(t *testing.T) {
	db := startPostgres(t)
	repo := database.NewPgUserRepo(db)

	user := &domain.User{
		Email:    "delete-me@example.com",
		Name:     "DeleteMe",
		Password: "hashed",
	}
	_ = repo.Create(context.Background(), user)

	if err := repo.Delete(context.Background(), user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	_, err := repo.GetByID(context.Background(), user.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

// ---------- Task Repository Tests ----------

func TestPgTaskRepoCreateAndGetByID(t *testing.T) {
	db := startPostgres(t)
	taskRepo := database.NewPgTaskRepo(db)
	userRepo := database.NewPgUserRepo(db)

	alice, _ := userRepo.GetByEmail(context.Background(), aliceEmail)

	task := &domain.Task{
		UserID:      alice.ID,
		Title:       "Integration task",
		Description: "Created during integration test",
		Status:      domain.TaskStatusTodo,
	}

	if err := taskRepo.Create(context.Background(), task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("expected task ID to be assigned")
	}

	got, err := taskRepo.GetByID(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Title != "Integration task" {
		t.Fatalf("expected title 'Integration task', got %q", got.Title)
	}
	if got.Status != domain.TaskStatusTodo {
		t.Fatalf("expected status todo, got %q", got.Status)
	}
}

func TestPgTaskRepoListByUserFixtures(t *testing.T) {
	db := startPostgres(t)
	taskRepo := database.NewPgTaskRepo(db)
	userRepo := database.NewPgUserRepo(db)

	alice, _ := userRepo.GetByEmail(context.Background(), aliceEmail)

	// fixtures.sql inserts 2 tasks for alice.
	tasks, err := taskRepo.ListByUser(context.Background(), alice.ID)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 fixture tasks for alice, got %d", len(tasks))
	}
}

func TestPgTaskRepoUpdate(t *testing.T) {
	db := startPostgres(t)
	taskRepo := database.NewPgTaskRepo(db)
	userRepo := database.NewPgUserRepo(db)

	bob, _ := userRepo.GetByEmail(context.Background(), bobEmail)
	tasks, _ := taskRepo.ListByUser(context.Background(), bob.ID)
	if len(tasks) == 0 {
		t.Fatal("expected fixture tasks for bob")
	}

	task := tasks[0]
	task.Status = domain.TaskStatusDone
	if err := taskRepo.Update(context.Background(), &task); err != nil {
		t.Fatalf("update task: %v", err)
	}

	got, _ := taskRepo.GetByID(context.Background(), task.ID)
	if got.Status != domain.TaskStatusDone {
		t.Fatalf("expected status done, got %q", got.Status)
	}
}

func TestPgTaskRepoDelete(t *testing.T) {
	db := startPostgres(t)
	taskRepo := database.NewPgTaskRepo(db)
	userRepo := database.NewPgUserRepo(db)

	alice, _ := userRepo.GetByEmail(context.Background(), aliceEmail)

	task := &domain.Task{
		UserID:      alice.ID,
		Title:       "To be deleted",
		Description: "",
		Status:      domain.TaskStatusTodo,
	}
	_ = taskRepo.Create(context.Background(), task)

	if err := taskRepo.Delete(context.Background(), task.ID); err != nil {
		t.Fatalf("delete task: %v", err)
	}

	_, err := taskRepo.GetByID(context.Background(), task.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

// ---------- Migration Test ----------

func TestMigrateIdempotent(t *testing.T) {
	db := startPostgres(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Migrations already ran via init scripts. Running again must not fail.
	if err := database.Migrate(ctx, db, logger); err != nil {
		t.Fatalf("second migration should be idempotent: %v", err)
	}
}

// ---------- Seed Test ----------

func TestSeedIdempotent(t *testing.T) {
	db := startPostgres(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Seed already loaded via init scripts. Running again must not fail
	// (ON CONFLICT DO NOTHING).
	if err := database.Seed(ctx, db, logger); err != nil {
		t.Fatalf("second seed should be idempotent: %v", err)
	}
}
