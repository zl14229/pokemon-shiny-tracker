package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/zl14229/pokemon-shiny-tracker/internal/files"
	"github.com/zl14229/pokemon-shiny-tracker/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Attachments struct{ DB *gorm.DB }
type LinkInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func attachmentOwner(db *gorm.DB, id uint64) error {
	var r model.ShinyRecord
	return db.First(&r, id).Error
}
func (s Attachments) store(ctx context.Context) (files.Store, error) {
	var dbs []struct {
		Name string
		File string
	}
	if e := s.DB.WithContext(ctx).Raw("PRAGMA database_list").Scan(&dbs).Error; e != nil {
		return files.Store{}, e
	}
	for _, d := range dbs {
		if d.Name == "main" && d.File != "" {
			return files.Store{Root: d.File + ".files"}, nil
		}
	}
	return files.Store{}, errors.New("file database required")
}
func (s Attachments) List(ctx context.Context, id uint64) ([]model.Attachment, error) {
	rows := []model.Attachment{}
	db := s.DB.WithContext(ctx)
	if e := attachmentOwner(db, id); e != nil {
		return rows, e
	}
	e := db.Where("shiny_record_id = ?", id).Order("id DESC").Find(&rows).Error
	return rows, e
}
func attachmentCapacity(tx *gorm.DB, id uint64) error {
	if e := attachmentOwner(tx, id); e != nil {
		return e
	}
	var n int64
	if e := tx.Model(&model.Attachment{}).Where("shiny_record_id = ?", id).Count(&n).Error; e != nil {
		return e
	}
	if n >= 100 {
		return conflict("每个个体最多100个有效附件或链接")
	}
	return nil
}
func (s Attachments) Upload(ctx context.Context, id uint64, name string, data []byte) (model.Attachment, error) {
	var a model.Attachment
	if _, e := files.Validate(name, data); e != nil {
		return a, invalid(e.Error())
	}
	if e := attachmentOwner(s.DB.WithContext(ctx), id); e != nil {
		return a, e
	}
	store, e := s.store(ctx)
	if e != nil {
		return a, e
	}
	info, e := store.Put(name, data)
	if e != nil {
		return a, e
	}
	a = model.Attachment{ShinyRecordID: id, Kind: "file", Name: name, MediaType: info.MediaType, StorageKey: &info.Key, SHA256: &info.SHA256, SizeBytes: &info.Size}
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := attachmentCapacity(tx, id); e != nil {
			return e
		}
		return tx.Omit(clause.Associations).Create(&a).Error
	})
	if e != nil {
		_ = store.RemoveNew(info.Key)
	}
	return a, e
}
func (s Attachments) Link(ctx context.Context, id uint64, in LinkInput) (model.Attachment, error) {
	var a model.Attachment
	in.Name = strings.TrimSpace(in.Name)
	u, e := url.Parse(in.URL)
	if e != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || len(in.URL) > 2048 || !utf8.ValidString(in.URL) || len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 200 {
		return a, invalid("链接须为无认证信息的HTTP/HTTPS地址，且填写名称")
	}
	a = model.Attachment{ShinyRecordID: id, Kind: "link", Name: in.Name, URL: &in.URL}
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := attachmentCapacity(tx, id); e != nil {
			return e
		}
		return tx.Omit(clause.Associations).Create(&a).Error
	})
	return a, e
}
func (s Attachments) Get(ctx context.Context, id uint64) (model.Attachment, error) {
	var a model.Attachment
	db := s.DB.WithContext(ctx)
	e := db.First(&a, id).Error
	if e == nil {
		e = attachmentOwner(db, a.ShinyRecordID)
	}
	return a, e
}
func (s Attachments) Content(ctx context.Context, id uint64) (model.Attachment, []byte, error) {
	a, e := s.Get(ctx, id)
	if e != nil {
		return a, nil, e
	}
	if a.Kind != "file" || a.StorageKey == nil || a.SHA256 == nil || a.SizeBytes == nil {
		return a, nil, invalid("该记录不是文件附件")
	}
	store, e := s.store(ctx)
	if e != nil {
		return a, nil, e
	}
	data, e := store.Read(*a.StorageKey, *a.SHA256, *a.SizeBytes)
	return a, data, e
}
func (s Attachments) Delete(ctx context.Context, id uint64) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var a model.Attachment
		if e := tx.First(&a, id).Error; e != nil {
			return e
		}
		if e := attachmentOwner(tx, a.ShinyRecordID); e != nil {
			return e
		}
		return tx.Delete(&a).Error
	})
}
