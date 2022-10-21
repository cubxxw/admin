package views

import (
	"fmt"
	"strings"

	"github.com/goplaid/web"
	"github.com/goplaid/x/i18n"
	"github.com/goplaid/x/presets"
	v "github.com/goplaid/x/vuetify"
	"github.com/qor/qor5/activity"
	"github.com/qor/qor5/l10n"
	"github.com/sunfmin/reflectutils"
	h "github.com/theplant/htmlgo"
	"gorm.io/gorm"
)

const (
	Localize   = "l10n_Localize"
	DoLocalize = "l10n_DoLocalize"
)

func registerEventFuncs(db *gorm.DB, mb *presets.ModelBuilder, lb *l10n.Builder, ab *activity.ActivityBuilder) {
	mb.RegisterEventFunc(Localize, localizeToConfirmation(lb))
	mb.RegisterEventFunc(DoLocalize, odLocalizeTo(db, mb))
}

func localizeToConfirmation(lb *l10n.Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		l10nMsgr := MustGetMessages(ctx.R)
		presetsMsgr := presets.MustGetMessages(ctx.R)

		id := ctx.R.FormValue(presets.ParamID)

		//todo get current locale
		fromLocale := lb.GetCorrectLocale(ctx.R)

		//todo search distinct locale_code except current locale
		toLocales := lb.GetSupportLocalesFromRequest(ctx.R)
		var toLocalesStrings []string
		for _, v := range toLocales {
			toLocalesStrings = append(toLocalesStrings, v.String())
		}

		r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
			Name: presets.DialogPortalName,
			Body: v.VDialog(
				v.VCard(
					v.VCardTitle(h.Text(l10nMsgr.Localize)),

					v.VCardText(
						h.Div(
							h.Div(
								h.Div(
									h.Label(l10nMsgr.LocalizeFrom).Class("v-label v-label--active theme--light").Style("left: 0px; right: auto; position: absolute;"),
									//h.Br(),
									h.Text(i18n.T(ctx.R, I10nLocalizeKey, lb.GetLocaleLabel(fromLocale))),
								).Class("v-text-field__slot"),
							).Class("v-input__slot"),
						).Class("v-input v-input--is-label-active v-input--is-dirty theme--light v-text-field v-text-field--is-booted"),
						v.VSelect().FieldName("localize_to").
							Label(l10nMsgr.LocalizeTo).
							Multiple(true).Chips(true).
							Items(toLocalesStrings),
					).Attr("style", "height: 200px;"),

					v.VCardActions(
						v.VSpacer(),
						v.VBtn(presetsMsgr.Cancel).
							Depressed(true).
							Class("ml-2").
							On("click", "vars.localizeConfirmation = false"),

						v.VBtn(presetsMsgr.OK).
							Color("primary").
							Depressed(true).
							Dark(true).
							Attr("@click", web.Plaid().
								EventFunc(DoLocalize).
								Query(presets.ParamID, id).
								Query("localize_from", lb.GetLocaleCode(fromLocale)).
								URL(ctx.R.URL.Path).
								Go()),
					),
				),
			).MaxWidth("600px").
				Attr("v-model", "vars.localizeConfirmation").
				Attr(web.InitContextVars, `{localizeConfirmation: false}`),
		})

		r.VarsScript = "setTimeout(function(){ vars.localizeConfirmation = true }, 100)"
		return
	}
}

func odLocalizeTo(db *gorm.DB, mb *presets.ModelBuilder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		segs := strings.Split(ctx.R.FormValue("id"), "_")
		id := segs[0]
		versionName := segs[1]

		to, exist := ctx.R.Form["localize_to"]
		if !exist {
			return
		}
		from := ctx.R.FormValue("localize_from")

		var obj = mb.NewModel()
		db.Unscoped().Where("id = ? AND version = ? AND locale_code = ?", id, versionName, from).First(&obj)

		me := mb.Editing()

		for _, toLocale := range to {
			obj := obj

			if err = reflectutils.Set(obj, "ID", id); err != nil {
				return
			}

			version := db.NowFunc().Format("2006-01-02")
			//var count int64
			//newObj := mb.NewModel()
			//db.Model(newObj).Unscoped().Where("id = ? AND version like ?", id, version+"%").Count(&count)

			versionName := fmt.Sprintf("%s-v%02v", version, 1)
			if err = reflectutils.Set(obj, "Version.Version", versionName); err != nil {
				return
			}
			if err = reflectutils.Set(obj, "Version.VersionName", versionName); err != nil {
				return
			}
			if err = reflectutils.Set(obj, "LocaleCode", toLocale); err != nil {
				return
			}
			fmt.Printf("%+v\n", obj)
			//if err = reflectutils.Set(obj, "Version.ParentVersion", segs[1]); err != nil {
			//	return
			//}

			if me.Validator != nil {
				if vErr := me.Validator(obj, ctx); vErr.HaveErrors() {
					me.UpdateOverlayContent(ctx, &r, obj, "", &vErr)
					return
				}
			}

			if err = db.Save(obj).Error; err != nil {
				return
			}

		}

		l10nMsgr := MustGetMessages(ctx.R)
		presets.ShowMessage(&r, l10nMsgr.SuccessfullyLocalized, "")

		// refresh current page
		r.Reload = true

		//r.PushState = web.Location(nil)
		return
	}
}
