package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/qor5/admin/v3/presets"
	"gorm.io/gorm"
)

// @snippet_begin(ActivityModelBuilder)
// a unique model builder is consist of typ and presetModel
type ModelBuilder struct {
	typ           reflect.Type                 // model type
	activity      *Builder                     // activity builder
	presetModel   *presets.ModelBuilder        // preset model builder
	skip          uint8                        // skip the refined data operator of the presetModel
	keys          []string                     // primary keys
	ignoredFields []string                     // ignored fields
	typeHandlers  map[reflect.Type]TypeHandler // type handlers
	link          func(any) string             // display the model link on the admin detail page
}

// @snippet_end

// AddKeys add keys to the model builder
func (mb *ModelBuilder) AddKeys(keys ...string) *ModelBuilder {
	for _, key := range keys {
		var find bool
		for _, mkey := range mb.keys {
			if mkey == key {
				find = true
				break
			}
		}
		if !find {
			mb.keys = append(mb.keys, key)
		}
	}
	return mb
}

// Keys set keys for the model builder
func (mb *ModelBuilder) Keys(keys ...string) *ModelBuilder {
	mb.keys = keys
	return mb
}

// LinkFunc set the link that linked to the modified record
func (mb *ModelBuilder) LinkFunc(f func(any) string) *ModelBuilder {
	mb.link = f
	return mb
}

// SkipCreate skip the created action for preset.ModelBuilder
func (mb *ModelBuilder) SkipCreate() *ModelBuilder {
	if mb.presetModel == nil {
		return mb
	}

	if mb.skip&Create == 0 {
		mb.skip |= Create
	}
	return mb
}

// SkipUpdate skip the update action for preset.ModelBuilder
func (mb *ModelBuilder) SkipUpdate() *ModelBuilder {
	if mb.presetModel == nil {
		return mb
	}

	if mb.skip&Update == 0 {
		mb.skip |= Update
	}
	return mb
}

// SkipDelete skip the delete action for preset.ModelBuilder
func (mb *ModelBuilder) SkipDelete() *ModelBuilder {
	if mb.presetModel == nil {
		return mb
	}

	if mb.skip&Delete == 0 {
		mb.skip |= Delete
	}
	return mb
}

// AddIgnoredFields append ignored fields to the default ignored fields, this would not overwrite the default ignored fields
func (mb *ModelBuilder) AddIgnoredFields(fields ...string) *ModelBuilder {
	mb.ignoredFields = append(mb.ignoredFields, fields...)
	return mb
}

// SetIgnoredFields set ignored fields to replace the default ignored fields with the new set.
func (mb *ModelBuilder) SetIgnoredFields(fields ...string) *ModelBuilder {
	mb.ignoredFields = fields
	return mb
}

// AddTypeHanders add type handers for the model builder
func (mb *ModelBuilder) AddTypeHanders(v any, f TypeHandler) *ModelBuilder {
	if mb.typeHandlers == nil {
		mb.typeHandlers = map[reflect.Type]TypeHandler{}
	}
	mb.typeHandlers[reflect.Indirect(reflect.ValueOf(v)).Type()] = f
	return mb
}

// KeysValue get model keys value
func (mb *ModelBuilder) KeysValue(v any) string {
	var (
		stringBuilder = strings.Builder{}
		reflectValue  = reflect.Indirect(reflect.ValueOf(v))
		reflectType   = reflectValue.Type()
	)

	for _, key := range mb.keys {
		if fields, ok := reflectType.FieldByName(key); ok {
			if reflectValue.FieldByName(key).IsZero() {
				continue
			}
			if fields.Anonymous {
				stringBuilder.WriteString(fmt.Sprintf("%v:", reflectValue.FieldByName(key).FieldByName(key).Interface()))
			} else {
				stringBuilder.WriteString(fmt.Sprintf("%v:", reflectValue.FieldByName(key).Interface()))
			}
		}
	}

	return strings.TrimRight(stringBuilder.String(), ":")
}

// AddRecords add records log
func (mb *ModelBuilder) AddRecords(action string, ctx context.Context, vs ...any) error {
	if len(vs) == 0 {
		return errors.New("data are empty")
	}

	var (
		creator = mb.activity.getCreatorFromContext(ctx)
		db      = mb.activity.getDBFromContext(ctx)
	)

	switch action {
	case ActivityView:
		for _, v := range vs {
			err := mb.AddViewRecord(creator, v, db)
			if err != nil {
				return err
			}
		}

	case ActivityDelete:
		for _, v := range vs {
			err := mb.AddDeleteRecord(creator, v, db)
			if err != nil {
				return err
			}
		}

	case ActivityCreate:
		for _, v := range vs {
			err := mb.AddCreateRecord(creator, v, db)
			if err != nil {
				return err
			}
		}

	case ActivityEdit:
		for _, v := range vs {
			err := mb.AddEditRecord(creator, v, db)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AddCustomizedRecord add customized record
func (mb *ModelBuilder) AddCustomizedRecord(action string, diff bool, ctx context.Context, obj any) error {
	var (
		creator = mb.activity.getCreatorFromContext(ctx)
		db      = mb.activity.getDBFromContext(ctx)
	)

	if !diff {
		return mb.save(creator, action, obj, db, "")
	}

	old, ok := findOld(obj, db)
	if !ok {
		return fmt.Errorf("can't find old data for %+v ", obj)
	}
	return mb.addDiff(action, creator, old, obj, db)
}

// AddViewRecord add view record
func (mb *ModelBuilder) AddViewRecord(creator any, v any, db *gorm.DB) error {
	return mb.save(creator, ActivityView, v, db, "")
}

// AddDeleteRecord	add delete record
func (mb *ModelBuilder) AddDeleteRecord(creator any, v any, db *gorm.DB) error {
	return mb.save(creator, ActivityDelete, v, db, "")
}

// AddSaverRecord will save a create log or a edit log
func (mb *ModelBuilder) AddSaveRecord(creator any, now any, db *gorm.DB) error {
	old, ok := findOld(now, db)
	if !ok {
		return mb.AddCreateRecord(creator, now, db)
	}
	return mb.AddEditRecordWithOld(creator, old, now, db)
}

// AddCreateRecord add create record
func (mb *ModelBuilder) AddCreateRecord(creator any, v any, db *gorm.DB) error {
	return mb.save(creator, ActivityCreate, v, db, "")
}

// AddEditRecord add edit record
func (mb *ModelBuilder) AddEditRecord(creator any, now any, db *gorm.DB) error {
	old, ok := findOld(now, db)
	if !ok {
		return fmt.Errorf("can't find old data for %+v ", now)
	}
	return mb.AddEditRecordWithOld(creator, old, now, db)
}

// AddEditRecord add edit record
func (mb *ModelBuilder) AddEditRecordWithOld(creator any, old, now any, db *gorm.DB) error {
	return mb.addDiff(ActivityEdit, creator, old, now, db)
}

func (mb *ModelBuilder) addDiff(action string, creator, old, now any, db *gorm.DB) error {
	diffs, err := mb.Diff(old, now)
	if err != nil {
		return err
	}

	if len(diffs) == 0 {
		return nil
	}

	b, err := json.Marshal(diffs)
	if err != nil {
		return err
	}

	return mb.save(creator, ActivityEdit, now, db, string(b))
}

// Diff get diffs between old and now value
func (mb *ModelBuilder) Diff(old, now any) ([]Diff, error) {
	return NewDiffBuilder(mb).Diff(old, now)
}

func (mb *ModelBuilder) save(creator any, action string, v any, db *gorm.DB, diffs string) error {
	log := &ActivityLog{}

	log.CreatedAt = time.Now()
	switch user := creator.(type) {
	case string:
		log.Creator = user
	case CreatorInterface:
		log.Creator = user.GetName()
		log.UserID = user.GetID()
	default:
		log.Creator = "unknown"
	}

	log.Action = action
	log.ModelName = mb.typ.Name()
	log.ModelKeys = mb.KeysValue(v)

	if mb.presetModel != nil && mb.presetModel.Info().URIName() != "" {
		log.ModelLabel = mb.presetModel.Info().URIName()
	} else {
		log.ModelLabel = "-"
	}

	if f := mb.link; f != nil {
		log.ModelLink = f(v)
	}

	if diffs == "" && action == ActivityEdit {
		return nil
	}

	if action == ActivityEdit {
		log.ModelDiffs = diffs
	}

	err := db.Save(log).Error
	if err != nil {
		return err
	}
	return nil
}
