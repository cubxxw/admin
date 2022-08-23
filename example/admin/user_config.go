package admin

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/qor/qor5/note"
	"github.com/qor/qor5/role"
	"github.com/sunfmin/reflectutils"

	"github.com/goplaid/web"
	"github.com/goplaid/x/presets"
	. "github.com/goplaid/x/vuetify"
	v "github.com/goplaid/x/vuetifyx"

	"github.com/qor/qor5/example/models"
	h "github.com/theplant/htmlgo"
	"gorm.io/gorm"
)

func configUser(b *presets.Builder, db *gorm.DB) {
	user := b.Model(&models.User{})
	note.Configure(db, b, user)

	ed := user.Editing(
		"Actions",
		"Name",
		"OAuthProvider",
		"Account",
		"Password",
		"Company",
		"Roles",
		"Status",
	)

	ed.ValidateFunc(func(obj interface{}, ctx *web.EventContext) (err web.ValidationErrors) {
		u := obj.(*models.User)
		if u.Account == "" {
			err.FieldError("Account", "Email is required")
		}
		return
	})
	user.RegisterEventFunc("roles_selector", rolesSelector(db))
	user.RegisterEventFunc("eventUnlockUser", func(ctx *web.EventContext) (r web.EventResponse, err error) {
		uid := ctx.R.FormValue("id")
		u := models.User{}
		if err = db.Where("id = ?", uid).First(&u).Error; err != nil {
			return r, err
		}
		if err = u.UnlockUser(db, &models.User{}); err != nil {
			return r, err
		}
		ed.UpdateOverlayContent(ctx, &r, &u, "", nil)
		return r, nil
	})

	user.RegisterEventFunc("eventSendResetPasswordEmail", func(ctx *web.EventContext) (r web.EventResponse, err error) {
		uid := ctx.R.FormValue("id")
		u := models.User{}
		if err = db.Where("id = ?", uid).First(&u).Error; err != nil {
			return r, err
		}
		token, err := u.GenerateResetPasswordToken(db, &models.User{})
		if err != nil {
			return r, err
		}
		r.VarsScript = fmt.Sprintf(`alert("http://localhost:9500/auth/reset-password?id=%s&token=%s")`, uid, token)
		return r, nil
	})

	ed.Field("Actions").ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		var actionBtns h.HTMLComponents
		u := obj.(*models.User)

		actionBtns = append(actionBtns,
			VBtn("Send Reset Password Email").
				Color("primary").
				Attr("@click", web.Plaid().EventFunc("eventSendResetPasswordEmail").
					Query("id", u.ID).Go()),
		)

		if u.GetLocked() {
			actionBtns = append(actionBtns,
				VBtn("Unlock").Color("primary").
					Attr("@click", web.Plaid().EventFunc("eventUnlockUser").
						Query("id", u.ID).Go(),
					),
			)
		}

		if len(actionBtns) == 0 {
			return nil
		}
		return h.Div(
			actionBtns...,
		).Class("mb-5 text-right")
	})

	ed.Field("Account").Label("Email").ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		return VTextField().
			FieldName(field.Name).
			Label(field.Label).
			Value(field.Value(obj)).
			ErrorMessages(field.Errors...)
	}).SetterFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) (err error) {
		u := obj.(*models.User)
		email := ctx.R.FormValue(field.Name)
		u.Account = email
		u.OAuthIndentifier = email
		return nil
	})

	ed.Field("OAuthProvider").Label("OAuthProvider").ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		return VSelect().FieldName(field.Name).
			Label(field.Label).Value(field.Value(obj)).
			Items([]string{"google", "microsoftonline"})
	})

	ed.Field("Password").ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		// TODO: polish UI
		return VTextField().
			FieldName(field.Name).
			Label(field.Label).
			Type("password")
	}).SetterFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) (err error) {
		u := obj.(*models.User)
		if v := ctx.R.FormValue(field.Name); v != "" {
			u.Password = v
			u.EncryptPassword()
		}
		return nil
	})

	ed.Field("Roles").
		ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
			var selectedItems = []DefaultOptionItem{}
			var values = []string{}
			u, ok := obj.(*models.User)
			if ok {
				var roles []role.Role
				db.Model(u).Association("Roles").Find(&roles)
				for _, r := range roles {
					values = append(values, fmt.Sprint(r.ID))
					selectedItems = append(selectedItems, DefaultOptionItem{
						Text:  r.Name,
						Value: fmt.Sprint(r.ID),
					})
				}
			}

			return v.VXAutocomplete().Label(field.Label).
				// ItemText("text").ItemValue("value").
				FieldName(field.Name).
				Multiple(true).Chips(true).Clearable(true).DeletableChips(true).
				Value(values).
				SelectedItems(selectedItems).
				// Items(items).
				CacheItems(true).
				ItemsEventFunc("roles_selector")
		}).
		SetterFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) (err error) {
			u, ok := obj.(*models.User)
			if !ok {
				return
			}
			rids := ctx.R.Form[field.Name]
			var roles []role.Role
			for _, id := range rids {
				uid, err1 := strconv.Atoi(id)
				if err1 != nil {
					continue
				}
				roles = append(roles, role.Role{
					Model: gorm.Model{ID: uint(uid)},
				})
			}

			if u.ID == 0 {
				err = reflectutils.Set(obj, field.Name, roles)
			} else {
				err = db.Model(u).Association(field.Name).Replace(roles)
			}
			if err != nil {
				return
			}
			return
		})

	ed.Field("Status").
		ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
			return VSelect().FieldName(field.Name).
				Label(field.Label).Value(field.Value(obj)).
				Items([]string{"active", "inactive"})
		})

	cl := user.Listing("ID", "Name", "Account", "Status", "Notes").PerPage(10)
	cl.Field("Account").Label("Email")

	cl.FilterDataFunc(func(ctx *web.EventContext) v.FilterData {
		return []*v.FilterItem{
			{
				Key:          "created",
				Label:        "Create Time",
				ItemType:     v.ItemTypeDate,
				SQLCondition: `cast(strftime('%%s', created_at) as INTEGER) %s ?`,
			},
			{
				Key:          "name",
				Label:        "Name",
				ItemType:     v.ItemTypeString,
				SQLCondition: `name %s ?`,
			},
			{
				Key:          "status",
				Label:        "Status",
				ItemType:     v.ItemTypeSelect,
				SQLCondition: `status %s ?`,
				Options: []*v.SelectItem{
					{Text: "Active", Value: "active"},
					{Text: "Inactive", Value: "inactive"},
				},
			},
		}
	})

	cl.FilterTabsFunc(func(ctx *web.EventContext) []*presets.FilterTab {
		return []*presets.FilterTab{
			{
				Label: "Felix",
				Query: url.Values{"name.ilike": []string{"felix"}},
			},
			{
				Label: "Active",
				Query: url.Values{"status": []string{"active"}},
			},
			{
				Label: "All",
				Query: url.Values{"all": []string{"1"}},
			},
		}
	})
}

func rolesSelector(db *gorm.DB) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		var roles []role.Role
		var items []DefaultOptionItem
		searchKey := ctx.R.FormValue("keyword")
		sql := db.Order("name").Limit(3)
		if searchKey != "" {
			sql = sql.Where("name ILIKE ?", fmt.Sprintf("%%%s%%", searchKey))
		}
		sql.Find(&roles)
		for _, r := range roles {
			items = append(items, DefaultOptionItem{
				Text:  r.Name,
				Value: fmt.Sprint(r.ID),
			})
		}
		r.Data = items
		return
	}
}
