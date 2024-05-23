package pagebuilder

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/qor5/admin/v3/l10n"
	"github.com/qor5/admin/v3/publish"
	"github.com/qor5/admin/v3/seo"
	"gorm.io/gorm"
)

type Page struct {
	gorm.Model
	Title      string
	Slug       string
	CategoryID uint

	SEO seo.Setting
	publish.Status
	publish.Schedule
	publish.Version
	l10n.Locale
}

func (p *Page) GetID() uint {
	return p.ID
}

func (*Page) TableName() string {
	return "page_builder_pages"
}

var l10nON bool

func (p *Page) L10nON() {
	l10nON = true
	return
}

func (p *Page) PrimarySlug() string {
	if !l10nON {
		return fmt.Sprintf("%v_%v", p.ID, p.Version.Version)
	}
	return fmt.Sprintf("%v_%v_%v", p.ID, p.Version.Version, p.LocaleCode)
}

func (p *Page) PrimaryColumnValuesBySlug(slug string) map[string]string {
	segs := strings.Split(slug, "_")
	if !l10nON {
		if len(segs) != 2 {
			panic("wrong slug")
		}

		return map[string]string{
			"id":                segs[0],
			publish.SlugVersion: segs[1],
		}
	}
	if len(segs) != 3 {
		panic("wrong slug")
	}

	return map[string]string{
		"id":                segs[0],
		publish.SlugVersion: segs[1],
		l10n.SlugLocaleCode: segs[2],
	}
}

func (p *Page) PermissionRN() []string {
	rn := []string{"pages", strconv.Itoa(int(p.ID)), p.Version.Version}
	if l10nON {
		rn = append(rn, p.LocaleCode)
	}
	return rn
}

func (p *Page) GetCategory(db *gorm.DB) (category Category, err error) {
	err = db.Where("id = ? AND locale_code = ?", p.CategoryID, p.LocaleCode).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	return
}

type Category struct {
	gorm.Model
	Name        string
	Path        string
	Description string

	IndentLevel int `gorm:"-"`

	l10n.Locale
}

func (c *Category) PrimarySlug() string {
	if !l10nON {
		return fmt.Sprintf("%v", c.ID)
	}
	return fmt.Sprintf("%v_%v", c.ID, c.LocaleCode)
}

func (c *Category) PrimaryColumnValuesBySlug(slug string) map[string]string {
	segs := strings.Split(slug, "_")
	if !l10nON {
		if len(segs) != 1 {
			panic("wrong slug")
		}

		return map[string]string{
			"id": segs[0],
		}
	}
	if len(segs) != 2 {
		panic("wrong slug")
	}

	return map[string]string{
		"id":                segs[0],
		l10n.SlugLocaleCode: segs[1],
	}
}

func (*Category) TableName() string {
	return "page_builder_categories"
}

type Container struct {
	gorm.Model
	PageID       uint
	PageVersion  string
	ModelName    string
	ModelID      uint
	DisplayOrder float64
	Shared       bool
	Hidden       bool
	DisplayName  string

	l10n.Locale
	LocalizeFromModelID uint
}

func (c *Container) PrimarySlug() string {
	if !l10nON {
		return fmt.Sprintf("%v", c.ID)
	}
	return fmt.Sprintf("%v_%v", c.ID, c.LocaleCode)
}

func (c *Container) PrimaryColumnValuesBySlug(slug string) map[string]string {
	segs := strings.Split(slug, "_")
	if !l10nON {
		if len(segs) != 1 {
			panic("wrong slug")
		}

		return map[string]string{
			"id": segs[0],
		}
	}
	if len(segs) != 2 {
		panic("wrong slug")
	}

	return map[string]string{
		"id":          segs[0],
		"locale_code": segs[1],
	}
}

func (*Container) TableName() string {
	return "page_builder_containers"
}

type DemoContainer struct {
	gorm.Model
	ModelName string
	ModelID   uint

	l10n.Locale
}

func (c *DemoContainer) PrimarySlug() string {
	if !l10nON {
		return fmt.Sprintf("%v", c.ID)
	}
	return fmt.Sprintf("%v_%v", c.ID, c.LocaleCode)
}

func (c *DemoContainer) PrimaryColumnValuesBySlug(slug string) map[string]string {
	segs := strings.Split(slug, "_")
	if !l10nON {
		if len(segs) != 1 {
			panic("wrong slug")
		}

		return map[string]string{
			"id": segs[0],
		}
	}
	if len(segs) != 2 {
		panic("wrong slug")
	}

	return map[string]string{
		"id":          segs[0],
		"locale_code": segs[1],
	}
}

func (*DemoContainer) TableName() string {
	return "page_builder_demo_containers"
}

type Template struct {
	gorm.Model
	Name        string
	Description string

	l10n.Locale
}

func (t *Template) GetID() uint {
	return t.ID
}

func (t *Template) PrimarySlug() string {
	if !l10nON {
		return fmt.Sprintf("%v", t.ID)
	}
	return fmt.Sprintf("%v_%v", t.ID, t.LocaleCode)
}

func (t *Template) PrimaryColumnValuesBySlug(slug string) map[string]string {
	segs := strings.Split(slug, "_")
	if !l10nON {
		if len(segs) != 1 {
			panic("wrong slug")
		}

		return map[string]string{
			"id": segs[0],
		}
	}
	if len(segs) != 2 {
		panic("wrong slug")
	}

	return map[string]string{
		"id":          segs[0],
		"locale_code": segs[1],
	}
}

func (*Template) TableName() string {
	return "page_builder_templates"
}

const templateVersion = "tpl"

func (t *Template) Page() *Page {
	return &Page{
		Model: t.Model,
		Title: t.Name,
		Slug:  "",
		Status: publish.Status{
			Status:    publish.StatusDraft,
			OnlineUrl: "",
		},
		Schedule: publish.Schedule{},
		Version: publish.Version{
			Version: templateVersion,
		},
		Locale: t.Locale,
	}
}
