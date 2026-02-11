package models

type URLShortener struct {
	ShortKey  string `gorm:"column:shortkey;type:varchar(10);primaryKey;not null"`
	URL       string `gorm:"column:url;type:varchar(2048);unique;not null"`
	ValidFrom int64  `gorm:"column:validfrom"`
	ValidTill int64  `gorm:"column:validtill"`
}

// TableName overrides the default (url_shorteners) to match your table
func (URLShortener) TableName() string {
	return "url_shortener"
}
