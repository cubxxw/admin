package activity

import (
	"gorm.io/gorm"
	"time"
)

const (
	ActivityView   = "View"
	ActivityEdit   = "Edit"
	ActivityCreate = "Create"
	ActivityDelete = "Delete"
)

type CreatorInterface interface {
	GetID() uint
	GetName() string
}

//TODO: Reconfiguration interface

type ActivityLogInterface interface {
	SetCreatedAt(time.Time)
	GetCreatedAt() time.Time
	SetUserID(uint)
	GetUserID() uint
	SetCreator(string)
	GetCreator() string
	SetAction(string)
	GetAction() string
	SetModelKeys(string)
	GetModelKeys() string
	SetModelName(string)
	GetModelName() string
	SetModelLabel(string)
	GetModelLabel() string
	SetModelLink(string)
	GetModelLink() string
	SetModelDiffs(string)
	GetModelDiffs() string
}

type ActivityLog struct {
	gorm.Model

	UserID     uint
	Creator    string
	Action     string
	ModelKeys  string `gorm:"index"`
	ModelName  string `gorm:"index"`
	ModelLabel string

	ModelLink  string
	ModelDiffs string `sql:"type:text;"`

	Comment string `gorm:"type:text;"`
}

func (al *ActivityLog) SetCreatedAt(t time.Time) {
	al.CreatedAt = t
}

func (al ActivityLog) GetCreatedAt() time.Time {
	return al.CreatedAt
}

func (al *ActivityLog) SetUserID(id uint) {
	al.UserID = id
}

func (al ActivityLog) GetUserID() uint {
	return al.UserID
}

func (al *ActivityLog) SetCreator(s string) {
	al.Creator = s
}

func (al *ActivityLog) GetCreator() string {
	return al.Creator
}

func (al *ActivityLog) SetAction(s string) {
	al.Action = s
}

func (al *ActivityLog) GetAction() string {
	return al.Action
}

func (al *ActivityLog) SetModelKeys(s string) {
	al.ModelKeys = s
}

func (al *ActivityLog) GetModelKeys() string {
	return al.ModelKeys
}

func (al *ActivityLog) SetModelName(s string) {
	al.ModelName = s
}

func (al *ActivityLog) GetModelName() string {
	return al.ModelName
}

func (al *ActivityLog) SetModelLabel(s string) {
	al.ModelLabel = s
}

func (al *ActivityLog) GetModelLabel() string {
	if al.ModelLabel == "" {
		return "-"
	}
	return al.ModelLabel
}

func (al *ActivityLog) SetModelLink(s string) {
	al.ModelLink = s
}

func (al *ActivityLog) GetModelLink() string {
	return al.ModelLink
}

func (al *ActivityLog) SetModelDiffs(s string) {
	al.ModelDiffs = s
}

func (al *ActivityLog) GetModelDiffs() string {
	return al.ModelDiffs
}
