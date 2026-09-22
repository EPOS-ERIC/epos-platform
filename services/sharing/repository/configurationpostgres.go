package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/sharing/domain"
	configurationsqlc "github.com/EPOS-ERIC/epos-platform/services/sharing/repository/sqlc"
	"github.com/google/uuid"
)

// ConfigurationRecord is the persistence-facing representation of a
// configuration. Transport and domain types map explicitly to and from this
// struct at the boundaries.
type ConfigurationRecord struct {
	ID             uuid.UUID
	Configuration  json.RawMessage
	Title          *string
	Description    *string
	ListOfServices []string
	IsOfficial     bool
	IsPublic       bool
	OwnerID        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ListFilter scopes which configurations are visible. A nil CallerID means an
// anonymous caller (public configurations only). IncludeAll grants access to
// every configuration regardless of ownership (administrators).
type ListFilter struct {
	Limit      int
	Offset     int
	Official   *bool
	CallerID   *string
	IncludeAll bool
}

type ConfigurationRepository interface {
	Create(ctx context.Context, id uuid.UUID, ownerID string, input domain.ConfigurationInput) (ConfigurationRecord, error)
	GetByID(ctx context.Context, id uuid.UUID) (ConfigurationRecord, error)
	List(ctx context.Context, filter ListFilter) (records []ConfigurationRecord, total int64, err error)
	Update(ctx context.Context, id uuid.UUID, input domain.ConfigurationInput) (ConfigurationRecord, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ConfigurationPostgresRepository struct {
	db      *sql.DB
	queries *configurationsqlc.Queries
}

func NewConfigurationPostgresRepository(db *sql.DB) *ConfigurationPostgresRepository {
	return &ConfigurationPostgresRepository{db: db, queries: configurationsqlc.New(db)}
}

func (r *ConfigurationPostgresRepository) Create(ctx context.Context, id uuid.UUID, ownerID string, input domain.ConfigurationInput) (ConfigurationRecord, error) {
	services, err := marshalServices(input.ListOfServices)
	if err != nil {
		return ConfigurationRecord{}, fmt.Errorf("create configuration: %w", err)
	}
	configuration, err := r.queries.CreateConfiguration(ctx, configurationsqlc.CreateConfigurationParams{
		ID:             id,
		Configuration:  input.Configuration,
		Title:          toNullString(input.Title),
		Description:    toNullString(input.Description),
		ListOfServices: services,
		IsOfficial:     input.IsOfficial,
		IsPublic:       input.IsPublic,
		OwnerID:        ownerID,
	})
	if err != nil {
		return ConfigurationRecord{}, fmt.Errorf("create configuration: %w", err)
	}
	return toConfigurationRecord(configuration)
}

func (r *ConfigurationPostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (ConfigurationRecord, error) {
	configuration, err := r.queries.GetConfigurationByID(ctx, id)
	if err != nil {
		return ConfigurationRecord{}, err
	}
	return toConfigurationRecord(configurationsqlc.Configuration(configuration))
}

func (r *ConfigurationPostgresRepository) List(ctx context.Context, filter ListFilter) ([]ConfigurationRecord, int64, error) {
	params := configurationsqlc.ListConfigurationsParams{
		CallerID:   toNullString(filter.CallerID),
		IncludeAll: filter.IncludeAll,
		Official:   toNullBool(filter.Official),
		PageLimit:  int32(filter.Limit),
		PageOffset: int32(filter.Offset),
	}
	total, err := r.queries.CountConfigurations(ctx, configurationsqlc.CountConfigurationsParams{
		CallerID:   params.CallerID,
		IncludeAll: params.IncludeAll,
		Official:   params.Official,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count configurations: %w", err)
	}
	rows, err := r.queries.ListConfigurations(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list configurations: %w", err)
	}
	records := make([]ConfigurationRecord, 0, len(rows))
	for _, row := range rows {
		record, err := toConfigurationRecord(configurationsqlc.Configuration(row))
		if err != nil {
			return nil, 0, fmt.Errorf("list configurations: %w", err)
		}
		records = append(records, record)
	}
	return records, total, nil
}

func (r *ConfigurationPostgresRepository) Update(ctx context.Context, id uuid.UUID, input domain.ConfigurationInput) (ConfigurationRecord, error) {
	services, err := marshalServices(input.ListOfServices)
	if err != nil {
		return ConfigurationRecord{}, fmt.Errorf("update configuration: %w", err)
	}
	configuration, err := r.queries.UpdateConfiguration(ctx, configurationsqlc.UpdateConfigurationParams{
		ID:             id,
		Configuration:  input.Configuration,
		Title:          toNullString(input.Title),
		Description:    toNullString(input.Description),
		ListOfServices: services,
		IsOfficial:     input.IsOfficial,
		IsPublic:       input.IsPublic,
	})
	if err != nil {
		return ConfigurationRecord{}, err
	}
	return toConfigurationRecord(configurationsqlc.Configuration(configuration))
}

func (r *ConfigurationPostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	affected, err := r.queries.DeleteConfiguration(ctx, id)
	if err != nil {
		return fmt.Errorf("delete configuration: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func toConfigurationRecord(configuration configurationsqlc.Configuration) (ConfigurationRecord, error) {
	services, err := unmarshalServices(configuration.ListOfServices)
	if err != nil {
		return ConfigurationRecord{}, fmt.Errorf("decode list_of_services: %w", err)
	}
	return ConfigurationRecord{
		ID:             configuration.ID,
		Configuration:  json.RawMessage(configuration.Configuration),
		Title:          fromNullString(configuration.Title),
		Description:    fromNullString(configuration.Description),
		ListOfServices: services,
		IsOfficial:     configuration.IsOfficial,
		IsPublic:       configuration.IsPublic,
		OwnerID:        configuration.OwnerID,
		CreatedAt:      configuration.CreatedAt,
		UpdatedAt:      configuration.UpdatedAt,
	}, nil
}

func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func fromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func toNullBool(value *bool) sql.NullBool {
	if value == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *value, Valid: true}
}

// marshalServices encodes the service list as a JSON array. Nil maps to an
// empty array so the NOT NULL column never receives NULL; absent and empty
// both mean "no services".
func marshalServices(values []string) (json.RawMessage, error) {
	if values == nil {
		return json.RawMessage("[]"), nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("encode list_of_services: %w", err)
	}
	return encoded, nil
}

func unmarshalServices(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}
