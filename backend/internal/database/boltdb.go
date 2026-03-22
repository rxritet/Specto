package database

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.etcd.io/bbolt"

	"github.com/rxritet/Specto/internal/domain"
)

// Bucket names used in BoltDB.
var (
	bucketUsers = []byte("users")
	bucketTasks = []byte("tasks")
)

// ---------- Connection ----------

// OpenBolt opens (or creates) a BoltDB file and ensures the required buckets exist.
func OpenBolt(path string, logger *slog.Logger) (*bbolt.DB, error) {
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("bolt open: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketUsers, bucketTasks} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bolt init buckets: %w", err)
	}

	logger.Info("opened boltdb", "path", path)
	return db, nil
}

// itob converts an int64 to an 8-byte big-endian slice (used as BoltDB keys).
func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// ---------- User Repository ----------

// BoltUserRepo implements domain.UserRepository backed by BoltDB.
type BoltUserRepo struct {
	db *bbolt.DB
}

func NewBoltUserRepo(db *bbolt.DB) *BoltUserRepo {
	return &BoltUserRepo{db: db}
}

func (r *BoltUserRepo) Create(_ context.Context, user *domain.User) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketUsers)

		id, _ := b.NextSequence()
		user.ID = int64(id)

		now := time.Now()
		user.CreatedAt = now
		user.UpdatedAt = now

		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return b.Put(itob(user.ID), data)
	})
}

func (r *BoltUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	var user domain.User
	err := r.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(bucketUsers).Get(itob(id))
		if data == nil {
			return fmt.Errorf("user %d not found", id)
		}
		return json.Unmarshal(data, &user)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *BoltUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	var found *domain.User
	err := r.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketUsers).ForEach(func(k, v []byte) error {
			var u domain.User
			if err := json.Unmarshal(v, &u); err != nil {
				return err
			}
			if u.Email == email {
				found = &u
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("user with email %q not found", email)
	}
	return found, nil
}

func (r *BoltUserRepo) Update(_ context.Context, user *domain.User) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketUsers)
		if b.Get(itob(user.ID)) == nil {
			return fmt.Errorf("user %d not found", user.ID)
		}

		user.UpdatedAt = time.Now()
		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return b.Put(itob(user.ID), data)
	})
}

func (r *BoltUserRepo) Delete(_ context.Context, id int64) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketUsers).Delete(itob(id))
	})
}

// ---------- Task Repository ----------

// BoltTaskRepo implements domain.TaskRepository backed by BoltDB.
type BoltTaskRepo struct {
	db *bbolt.DB
}

func NewBoltTaskRepo(db *bbolt.DB) *BoltTaskRepo {
	return &BoltTaskRepo{db: db}
}

func (r *BoltTaskRepo) Create(_ context.Context, task *domain.Task) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTasks)

		id, _ := b.NextSequence()
		task.ID = int64(id)

		now := time.Now()
		task.CreatedAt = now
		task.UpdatedAt = now

		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		return b.Put(itob(task.ID), data)
	})
}

func (r *BoltTaskRepo) GetByID(_ context.Context, id int64) (*domain.Task, error) {
	var task domain.Task
	err := r.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(bucketTasks).Get(itob(id))
		if data == nil {
			return fmt.Errorf("task %d not found", id)
		}
		return json.Unmarshal(data, &task)
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *BoltTaskRepo) ListByUser(_ context.Context, userID int64) ([]domain.Task, error) {
	var tasks []domain.Task
	err := r.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketTasks).ForEach(func(k, v []byte) error {
			var t domain.Task
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			if t.UserID == userID {
				tasks = append(tasks, t)
			}
			return nil
		})
	})
	return tasks, err
}

func (r *BoltTaskRepo) Update(_ context.Context, task *domain.Task) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		if b.Get(itob(task.ID)) == nil {
			return fmt.Errorf("task %d not found", task.ID)
		}

		task.UpdatedAt = time.Now()
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		return b.Put(itob(task.ID), data)
	})
}

func (r *BoltTaskRepo) Delete(_ context.Context, id int64) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketTasks).Delete(itob(id))
	})
}
