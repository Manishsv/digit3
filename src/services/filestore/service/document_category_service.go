package service

import (
	"gin/models"
	"gin/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentCategoryService struct {
	repo repository.ArtifactRepository
}

func NewDocumentCategoryService(repo repository.ArtifactRepository) *DocumentCategoryService {
	return &DocumentCategoryService{repo: repo}
}

func (s *DocumentCategoryService) CreateDocumentCategory(doc models.DocumentCategory, tenantId string) (models.DocumentCategory, error) {
	doc.TenantId = tenantId
	doc.ID = uint64(uuid.New().ID())
	return s.repo.CreateDocumentCategory(doc, tenantId)
}

func (s *DocumentCategoryService) SearchDocumentCategories(docType string, docCode string, isSensitive string, tenantId string) ([]models.DocumentCategory, error) {
	return s.repo.SearchDocumentCategories(docType, docCode, isSensitive, tenantId)
}

func (s *DocumentCategoryService) GetDocumentCategoryByCode(docCode string, tenantId string) (models.DocumentCategory, error) {
	// Call the repository's SearchDocumentCategories with only code and tenantId
	docs, err := s.repo.SearchDocumentCategories("", docCode, "", tenantId)
	if err != nil {
		return models.DocumentCategory{}, err
	}
	if len(docs) == 0 {
		return models.DocumentCategory{}, gorm.ErrRecordNotFound
	}
	return docs[0], nil
}

func (s *DocumentCategoryService) UpdateDocumentCategory(ExistingDoc models.DocumentCategory, docCode string, tenantId string, NewDoc models.DocumentCategory) (models.DocumentCategory, error) {

	ExistingDoc.AllowedFormats = NewDoc.AllowedFormats
	ExistingDoc.IsActive = NewDoc.IsActive
	ExistingDoc.IsSensitive = NewDoc.IsActive
	ExistingDoc.MaxSize = NewDoc.MaxSize
	ExistingDoc.MinSize = NewDoc.MinSize

	// Call the repository update
	err := s.repo.UpdateDocumentCategory(ExistingDoc, docCode, tenantId)
	if err != nil {
		return models.DocumentCategory{}, err
	}
	return ExistingDoc, nil
}

func (s *DocumentCategoryService) DeleteDocumentCategory(docCode string, tenantId string) (models.DocumentCategory, error) {
	// Fetch the document category to return it after deletion
	doc, err := s.GetDocumentCategoryByCode(docCode, tenantId)
	if err != nil {
		return models.DocumentCategory{}, err
	}

	// Call the repository to delete the document category
	s.repo.DeleteDocumentCategory(docCode, tenantId)

	return doc, nil
}
