package views

import (
	"fmt"
	"github.com/goplaid/web"
	"github.com/goplaid/x/i18n"
	"github.com/goplaid/x/presets"
	. "github.com/goplaid/x/vuetify"
	"github.com/qor/qor5/publish"
	"github.com/sunfmin/reflectutils"
	h "github.com/theplant/htmlgo"
	"gorm.io/gorm"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func sidePanel(db *gorm.DB, mb *presets.ModelBuilder) presets.ComponentFunc {
	return func(ctx *web.EventContext) h.HTMLComponent {
		segs := strings.Split(ctx.Event.Params[0], "_")
		id := segs[0]

		c := h.Div()

		ov := VCard(
			VCardTitle(h.Text("Online Version")),
		)
		c.AppendChildren(ov)

		lv := map[string]interface{}{}
		db.Session(&gorm.Session{NewDB: true}).Model(mb.NewModel()).
			Where("id = ? AND status = 'Published'", id).
			First(&lv)
		if len(lv) > 0 {
			ov.AppendChildren(
				VSimpleTable(
					h.Tbody(
						h.Tr(
							h.Td(h.Button(fmt.Sprint(lv["version"]))),
							h.Td(h.Button(fmt.Sprint(lv["status"]))),
						).Attr("@click", web.Plaid().EventFunc(switchVersionEvent, fmt.Sprint(lv["id"]), fmt.Sprint(lv["version"])).Go()),
					),
				),
			)
		}

		c.AppendChildren(h.Br())

		versionsList := VCard(
			VCardTitle(h.Text("Versions List")),
		)
		c.AppendChildren(versionsList)

		var results []map[string]interface{}
		db.Session(&gorm.Session{NewDB: true}).Model(mb.NewModel()).
			Where("id = ?", id).Order("version DESC").
			Find(&results)

		tbody := h.Tbody()
		for _, r := range results {
			tr := h.Tr(
				h.Td(h.Button(fmt.Sprint(r["version"]))),
				h.Td(h.Button(fmt.Sprint(r["status"]))),
			).Attr("@click", web.Plaid().EventFunc(switchVersionEvent, fmt.Sprint(fmt.Sprint(r["id"])), fmt.Sprint(r["version"])).Go())
			tbody.AppendChildren(tr)
		}

		versionsList.AppendChildren(VSimpleTable(tbody))

		return c
	}
}

func switchVersionAction(db *gorm.DB, mb *presets.ModelBuilder, publisher *publish.Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		id := ctx.Event.Params[0]
		version := ctx.Event.Params[1]

		eb := mb.Editing()

		obj := mb.NewModel()
		obj, err = eb.Fetcher(obj, fmt.Sprintf("%v_%v", id, version), ctx)

		eb.UpdateRightDrawerContent(ctx, &r, obj, "", err)
		return
	}
}

func saveNewVersionAction(db *gorm.DB, mb *presets.ModelBuilder, publisher *publish.Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		segs := strings.Split(ctx.Event.Params[0], "_")
		id := segs[0]
		//version := ctx.Event.Params[1]

		var newObj = mb.NewModel()
		// don't panic for fields that set in SetterFunc
		_ = ctx.UnmarshalForm(newObj)

		var obj = mb.NewModel()

		me := mb.Editing()
		if me.Setter != nil {
			me.Setter(obj, ctx)
		}

		vErr := me.RunSetterFunc(ctx, &r, obj, newObj)
		if vErr.HaveErrors() {
			return
		}

		intID, err := strconv.Atoi(id)
		if err != nil {
			return
		}
		if err = reflectutils.Set(obj, "ID", uint(intID)); err != nil {
			return
		}

		version := time.Now().Format("2006-01-02")
		var count int64
		db.Model(newObj).Where("id = ? AND version like ?", id, version+"%").Count(&count)

		if err = reflectutils.Set(obj, "Version.Version", fmt.Sprintf("%s-v%v", version, count+1)); err != nil {
			return
		}

		if me.Validator != nil {
			if vErr := me.Validator(obj, ctx); vErr.HaveErrors() {
				me.UpdateRightDrawerContent(ctx, &r, obj, "", &vErr)
				return
			}
		}

		err1 := me.Saver(obj, ctx.Event.Params[0], ctx)
		if err1 != nil {
			me.UpdateRightDrawerContent(ctx, &r, obj, "", err1)
			return
		}

		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)
		me.UpdateRightDrawerContent(ctx, &r, obj, msgr.CreatedSuccessfully, nil)
		return
	}
}

func searcher(db *gorm.DB, mb *presets.ModelBuilder) presets.SearchFunc {
	return func(obj interface{}, params *presets.SearchParams, ctx *web.EventContext) (r interface{}, totalCount int, err error) {
		ilike := "ILIKE"
		if db.Dialector.Name() == "sqlite" {
			ilike = "LIKE"
		}

		wh := db.Model(obj)
		if len(params.KeywordColumns) > 0 && len(params.Keyword) > 0 {
			var segs []string
			var args []interface{}
			for _, c := range params.KeywordColumns {
				segs = append(segs, fmt.Sprintf("%s %s ?", c, ilike))
				args = append(args, fmt.Sprintf("%%%s%%", params.Keyword))
			}
			wh = wh.Where(strings.Join(segs, " OR "), args...)
		}

		for _, cond := range params.SQLConditions {
			wh = wh.Where(strings.Replace(cond.Query, " ILIKE ", " "+ilike+" ", -1), cond.Args...)
		}

		stmt := &gorm.Statement{DB: db}
		stmt.Parse(mb.NewModel())
		tn := stmt.Schema.Table

		var c int64
		sql := fmt.Sprintf("(%v.id, %v.version) IN (SELECT %v.id, MAX(%v.version) FROM %v GROUP BY %v.id)", tn, tn, tn, tn, tn, tn)
		if err = wh.Where(sql).Count(&c).Error; err != nil {
			return
		}
		totalCount = int(c)

		if params.PerPage > 0 {
			wh = wh.Limit(int(params.PerPage))
			page := params.Page
			if page == 0 {
				page = 1
			}
			offset := (page - 1) * params.PerPage
			wh = wh.Offset(int(offset))
		}

		orderBy := params.OrderBy
		if len(orderBy) > 0 {
			wh = wh.Order(orderBy)
		}

		if err = wh.Find(obj).Error; err != nil {
			return
		}
		r = reflect.ValueOf(obj).Elem().Interface()
		return
	}
}
