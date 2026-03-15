package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lokewate/go-workflow/state"
	"gorm.io/gorm"
)

// GormWorkflowInstance is the database model for WorkflowInstance.
type GormWorkflowInstance struct {
	ID         string `gorm:"primaryKey"`
	WorkflowID string
	Status     string
}

// GormWorkflowState is the database model for the instance state (data and tokens).
type GormWorkflowState struct {
	InstanceID string `gorm:"primaryKey"`
	Data       []byte
	Tokens     []byte
}

// DBRepo implements Repo using GORM.
type DBRepo struct {
	db *gorm.DB
}

// NewDBRepo initializes a new DBRepo and runs auto-migration.
func NewDBRepo(db *gorm.DB) (*DBRepo, error) {
	err := db.AutoMigrate(&GormWorkflowInstance{}, &GormWorkflowState{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate workflow tables: %w", err)
	}
	return &DBRepo{db: db}, nil
}

// NewContext creates a GlobalContext wired to this repo's persistence layer.
func (r *DBRepo) NewContext(instID string) state.GlobalContext {
	return state.NewMapContextWithID(
		instID,
		r.loadState(instID),
		r.saveState(instID),
	)
}

// Get retrieves a workflow instance by ID.
func (r *DBRepo) Get(ctx context.Context, id string) (*WorkflowInstance, error) {
	var gormInst GormWorkflowInstance
	if err := r.db.WithContext(ctx).First(&gormInst, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}

	inst := &WorkflowInstance{
		ID:         gormInst.ID,
		WorkflowID: gormInst.WorkflowID,
		Status:     WorkflowStatus(gormInst.Status),
	}

	// Wire context to the repo's persistence
	inst.Context = state.NewMapContext(
		r.loadState(id),
		r.saveState(id),
	)

	if err := inst.Context.Load(ctx, id); err != nil {
		return nil, err
	}

	return inst, nil
}

// Save persists a workflow instance metadata.
func (r *DBRepo) Save(ctx context.Context, inst *WorkflowInstance) error {
	gormInst := GormWorkflowInstance{
		ID:         inst.ID,
		WorkflowID: inst.WorkflowID,
		Status:     string(inst.Status),
	}
	return r.db.WithContext(ctx).Save(&gormInst).Error
}

func (r *DBRepo) loadState(_ string) func(string) (map[string]any, []state.Token, error) {
	return func(id string) (map[string]any, []state.Token, error) {
		var stateRow GormWorkflowState
		if err := r.db.First(&stateRow, "instance_id = ?", id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return make(map[string]any), []state.Token{}, nil
			}
			return nil, nil, err
		}

		var data map[string]any
		if len(stateRow.Data) > 0 {
			if err := json.Unmarshal(stateRow.Data, &data); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal state data: %w", err)
			}
		}

		var tokens []state.Token
		if len(stateRow.Tokens) > 0 {
			if err := json.Unmarshal(stateRow.Tokens, &tokens); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal tokens: %w", err)
			}
		}

		return data, tokens, nil
	}
}

func (r *DBRepo) saveState(_ string) func(string, map[string]any, []state.Token) error {
	return func(id string, data map[string]any, tokens []state.Token) error {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal state data: %w", err)
		}

		tokensBytes, err := json.Marshal(tokens)
		if err != nil {
			return fmt.Errorf("failed to marshal tokens: %w", err)
		}

		stateRow := GormWorkflowState{
			InstanceID: id,
			Data:       dataBytes,
			Tokens:     tokensBytes,
		}

		return r.db.Save(&stateRow).Error
	}
}
