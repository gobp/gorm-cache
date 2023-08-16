package cache

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
)

var (
	ErrMissingKey = errors.New("missing key on storage")
)

type Storage storage.Storage

type Serializer interface {
	Serialize(v any) ([]byte, error)

	Deserialize(data []byte, v any) error
}

type plugin struct {
	storage      Storage
	serializer   Serializer
	prefix       string
	expires      time.Duration
	keyGenerator func(string) string
}

func New(config ...Config) *plugin {
	// Set default config
	cfg := configDefault(config...)

	return &plugin{
		expires:      cfg.Expires,
		keyGenerator: cfg.KeyGenerator,
		storage:      cfg.Storage,
		prefix:       cfg.Prefix,
		serializer:   cfg.Serializer,
	}
}

func (p *plugin) Name() string {
	return "gobp:cache"
}

func (p *plugin) Initialize(tx *gorm.DB) error {
	// TODO: see all the callbacks that we can modify

	if err := tx.Callback().Query().Replace("gorm:query", p.Query); err != nil {
		return err
	}

	if err := tx.Callback().Query().Replace("gorm:row", p.RowQuery); err != nil {
		return err
	}

	return nil
}

func (p *plugin) RowQuery(tx *gorm.DB) {
	ctx := tx.Statement.Context

	var ttl time.Duration
	var hasTTL bool

	if ttl, hasTTL = FromExpiration(ctx); !hasTTL {
		tx.Logger.Info(ctx, "using default TTL")
		ttl = p.expires
	}

	var key string
	var hasKey bool

	identifier := buildIdentifier(tx)

	// Checks if there's a custom key
	if key, hasKey = FromKey(ctx); !hasKey {
		key = p.prefix + p.keyGenerator(identifier)
	}

	// Get from cached data
	if err := p.QueryCache(key, tx.Statement.Dest); err == nil {
		tx.Logger.Info(ctx, "from cache")
		return
	}

	// Get from database
	p.RowQueryDB(tx)
	if tx.Error != nil {
		return
	}
	tx.Logger.Info(ctx, "from database")

	// Persist to cache layer
	if err := p.SaveCache(key, tx.Statement.Dest, ttl); err != nil {
		tx.Logger.Error(ctx, err.Error())
		return
	}
	tx.Logger.Info(ctx, "cache persisted")
}

func (p *plugin) Query(tx *gorm.DB) {
	ctx := tx.Statement.Context

	var ttl time.Duration
	var hasTTL bool

	if ttl, hasTTL = FromExpiration(ctx); !hasTTL {
		tx.Logger.Info(ctx, "using default TTL")
		ttl = p.expires
	}

	var key string
	var hasKey bool

	identifier := buildIdentifier(tx)

	// Checks if there's a custom key
	if key, hasKey = FromKey(ctx); !hasKey {
		key = p.prefix + p.keyGenerator(identifier)
	}

	// Get from cached data
	if err := p.QueryCache(key, tx.Statement.Dest); err == nil {
		tx.Logger.Info(ctx, "from cache")
		return
	}

	// Get from database
	p.QueryDB(tx)
	if tx.Error != nil {
		return
	}
	tx.Logger.Info(ctx, "from database")

	// Persist to cache layer
	if err := p.SaveCache(key, tx.Statement.Dest, ttl); err != nil {
		tx.Logger.Error(ctx, err.Error())
		return
	}
	tx.Logger.Info(ctx, "cache persisted")

}

func buildIdentifier(db *gorm.DB) string {
	// Build query identifier,
	//	for that reason we need to compile all arguments into a string
	//	and concat them with the SQL query itself

	callbacks.BuildQuerySQL(db)

	var (
		identifier string
		query      string
		queryArgs  string
	)

	query = db.Statement.SQL.String()
	queryArgs = fmt.Sprintf("%v", db.Statement.Vars)
	identifier = fmt.Sprintf("%s-%s", query, queryArgs)

	return identifier
}

func (p *plugin) QueryDB(tx *gorm.DB) {
	// HACK: Rewrite the Query method here because we don't want to execute callbacks.BuildQuerySQL twice
	// HACK: Copied from https://github.com/go-gorm/gorm/blob/bae684b3639dff3e35d0ed330bc82c12e8282110/callbacks/query.go#L15-L31

	if tx.Error == nil {
		// callbacks.BuildQuerySQL(tx) // We don't want this line

		if !tx.DryRun && tx.Error == nil {
			rows, err := tx.Statement.ConnPool.QueryContext(tx.Statement.Context, tx.Statement.SQL.String(), tx.Statement.Vars...)
			if err != nil {
				tx.AddError(err)
				return
			}
			defer func() {
				tx.AddError(rows.Close())
			}()
			gorm.Scan(rows, tx, 0)
		}
	}
}

func (p *plugin) RowQueryDB(tx *gorm.DB) {
	// HACK: Rewrite the RowQueryDB method here because we don't want to execute callbacks.BuildQuerySQL twice
	// HACK: Copied from https://github.com/go-gorm/gorm/blob/bae684b3639dff3e35d0ed330bc82c12e8282110/callbacks/row.go#L7-L23

	if tx.Error == nil {
		// callbacks.BuildQuerySQL(tx) // We don't want this line

		if tx.DryRun || tx.Error != nil {
			return
		}

		if isRows, ok := tx.Get("rows"); ok && isRows.(bool) {
			tx.Statement.Settings.Delete("rows")
			tx.Statement.Dest, tx.Error = tx.Statement.ConnPool.QueryContext(tx.Statement.Context, tx.Statement.SQL.String(), tx.Statement.Vars...)
		} else {
			tx.Statement.Dest = tx.Statement.ConnPool.QueryRowContext(tx.Statement.Context, tx.Statement.SQL.String(), tx.Statement.Vars...)
		}

		tx.RowsAffected = -1
	}
}

func (p *plugin) QueryCache(key string, dest any) error {
	values, err := p.storage.Get(key)
	if err != nil {
		return err
	}

	if values == nil {
		return ErrMissingKey
	}

	// TODO: why ?
	switch dest.(type) {
	case *int64:
		dest = 0
	}

	return p.serializer.Deserialize(values, dest)
}

func (p *plugin) SaveCache(key string, dest any, ttl time.Duration) error {
	values, err := p.serializer.Serialize(dest)
	if err != nil {
		return err
	}

	return p.storage.Set(key, values, ttl)
}
