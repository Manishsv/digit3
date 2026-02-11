package models

import (
	"fmt"
)

// AbstractPersistable is a base struct for persistent entities
type AbstractPersistable struct {
	Version int64 `gorm:"column:version"`
}

// PersistableEntity is an interface that all persistent entities must implement
type PersistableEntity interface {
	GetID() interface{}
	SetID(id interface{})
	IsNew() bool
	String() string
}

// GetVersion returns the version number of the entity
func (a *AbstractPersistable) GetVersion() int64 {
	return a.Version
}

// SetVersion sets the version number of the entity
func (a *AbstractPersistable) SetVersion(version int64) {
	a.Version = version
}

// IsNew checks if the entity is new (has no ID)
func (a *AbstractPersistable) IsNew() bool {
	return a.GetID() == nil
}

// String returns a string representation of the entity
func (a *AbstractPersistable) String() string {
	return fmt.Sprintf("Entity of type %T with id: %v", a, a.GetID())
}

// GetID is an abstract method that must be implemented by concrete types
func (a *AbstractPersistable) GetID() interface{} {
	panic("GetID() must be implemented by concrete types")
}

// SetID is an abstract method that must be implemented by concrete types
func (a *AbstractPersistable) SetID(id interface{}) {
	panic("SetID() must be implemented by concrete types")
}

// ArtifactEntity represents the database entity for artifacts
type ArtifactEntity struct {
	AbstractPersistable
	ID               uint64 `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`
	FileStoreID      string `gorm:"column:filestoreid;type:varchar(36);uniqueIndex;not null"`
	FileName         string `gorm:"column:filename;not null"`
	ContentType      string `gorm:"column:contenttype"`
	Module           string `gorm:"column:module"`
	Tag              string `gorm:"column:tag"`
	TenantID         string `gorm:"column:tenantid"`
	FileSource       string `gorm:"column:filesource"`
	CreatedBy        string `gorm:"column:createdby"`
	LastModifiedBy   string `gorm:"column:lastmodifiedby"`
	CreatedTime      int64  `gorm:"column:createdtime"`
	LastModifiedTime int64  `gorm:"column:lastmodifiedtime"`
}

// TableName specifies the table name for ArtifactEntity
func (ArtifactEntity) TableName() string {
	return "eg_filestoremap_v2"
}

// GetID implements the PersistableEntity interface
func (a *ArtifactEntity) GetID() interface{} {
	return a.ID
}

// SetID implements the PersistableEntity interface
func (a *ArtifactEntity) SetID(id interface{}) {
	a.ID = id.(uint64)
}

// GetFileLocation converts the entity to a FileLocation model
func (a *ArtifactEntity) GetFileLocation() FileLocation {
	return FileLocation{
		FileStoreID: a.FileStoreID,
		Module:      a.Module,
		Tag:         a.Tag,
		TenantID:    a.TenantID,
		FileName:    a.FileName,
		FileSource:  a.FileSource,
	}
}

// NewArtifactEntity creates a new ArtifactEntity with the given parameters
func NewArtifactEntity(
	fileStoreID string,
	fileName string,
	contentType string,
	module string,
	tag string,
	tenantID string,
	fileSource string,
	createdBy string,
	lastModifiedBy string,
	createdTime int64,
	lastModifiedTime int64,
) *ArtifactEntity {
	return &ArtifactEntity{
		FileStoreID:      fileStoreID,
		FileName:         fileName,
		ContentType:      contentType,
		Module:           module,
		Tag:              tag,
		TenantID:         tenantID,
		FileSource:       fileSource,
		CreatedBy:        createdBy,
		LastModifiedBy:   lastModifiedBy,
		CreatedTime:      createdTime,
		LastModifiedTime: lastModifiedTime,
	}
}

// RequestInfo contains information about the request
type RequestInfo struct {
	UserID string
}
