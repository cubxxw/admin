package publish

import (
	"fmt"

	"github.com/qor5/admin/v3/activity"
	"github.com/qor5/admin/v3/presets"
	"github.com/qor5/admin/v3/presets/actions"
	"github.com/qor5/admin/v3/utils"
	v "github.com/qor5/ui/v3/vuetify"
	"github.com/qor5/web/v3"
	"github.com/qor5/x/v3/i18n"
	"github.com/sunfmin/reflectutils"
	h "github.com/theplant/htmlgo"
	"gorm.io/gorm"
)

const (
	PortalSchedulePublishDialog = "publish_PortalSchedulePublishDialog"
	PortalPublishCustomDialog   = "publish_PortalPublishCustomDialog"
)

func saveNewVersionAction(db *gorm.DB, mb *presets.ModelBuilder, _ *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		toObj := mb.NewModel()
		slugger := toObj.(presets.SlugDecoder)
		paramID := ctx.Param(presets.ParamID)
		currentVersionName := slugger.PrimaryColumnValuesBySlug(paramID)["version"]

		me := mb.Editing()
		vErr := me.RunSetterFunc(ctx, false, toObj)

		if vErr.HaveErrors() {
			me.UpdateOverlayContent(ctx, &r, toObj, "", &vErr)
			return
		}

		fromObj := mb.NewModel()
		utils.PrimarySluggerWhere(db, mb.NewModel(), paramID).First(fromObj)
		if err = utils.SetPrimaryKeys(fromObj, toObj, db, paramID); err != nil {
			return
		}

		if err = reflectutils.Set(toObj, "Version.ParentVersion", currentVersionName); err != nil {
			return
		}

		if me.Validator != nil {
			if vErr := me.Validator(toObj, ctx); vErr.HaveErrors() {
				me.UpdateOverlayContent(ctx, &r, toObj, "", &vErr)
				return
			}
		}

		if err = me.Saver(toObj, paramID, ctx); err != nil {
			me.UpdateOverlayContent(ctx, &r, toObj, "", err)
			return
		}

		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)
		presets.ShowMessage(&r, msgr.SuccessfullyCreated, "")

		if ctx.R.URL.Query().Get(presets.ParamInDialog) == "true" {
			web.AppendRunScripts(&r,
				"vars.presetsDialog = false",
				web.Plaid().
					URL(ctx.R.RequestURI).
					EventFunc(actions.UpdateListingDialog).
					StringQuery(ctx.R.URL.Query().Get(presets.ParamListingQueries)).
					Go(),
			)
		} else {
			r.Reload = true
		}

		return
	}
}

func duplicateVersionAction(db *gorm.DB, mb *presets.ModelBuilder, _ *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		toObj := mb.NewModel()
		slugger := toObj.(presets.SlugDecoder)
		paramID := ctx.Param(presets.ParamID)
		currentVersionName := slugger.PrimaryColumnValuesBySlug(paramID)["version"]
		me := mb.Editing()

		fromObj := mb.NewModel()
		if err = utils.PrimarySluggerWhere(db, mb.NewModel(), paramID).First(fromObj).Error; err != nil {
			return
		}
		if err = utils.SetPrimaryKeys(fromObj, toObj, db, paramID); err != nil {
			presets.ShowMessage(&r, err.Error(), "error")
			return
		}

		if vErr := me.SetObjectFields(fromObj, toObj, &presets.FieldContext{
			ModelInfo: mb.Info(),
		}, false, presets.ContextModifiedIndexesBuilder(ctx).FromHidden(ctx.R), ctx); vErr.HaveErrors() {
			presets.ShowMessage(&r, vErr.Error(), "error")
			return
		}

		if err = reflectutils.Set(toObj, "Version.ParentVersion", currentVersionName); err != nil {
			presets.ShowMessage(&r, err.Error(), "error")
			return
		}

		if me.Validator != nil {
			if vErr := me.Validator(toObj, ctx); vErr.HaveErrors() {
				presets.ShowMessage(&r, vErr.Error(), "error")
				return
			}
		}

		if err = me.Saver(toObj, paramID, ctx); err != nil {
			presets.ShowMessage(&r, err.Error(), "error")
			return
		}

		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)
		presets.ShowMessage(&r, msgr.SuccessfullyCreated, "")
		se := toObj.(presets.SlugEncoder)
		newQueries := ctx.Queries()
		newQueries.Del(presets.ParamID)
		r.PushState = web.Location(newQueries).URL(mb.Info().DetailingHref(se.PrimarySlug()))
		return
	}
}

func versionActionsFunc(m *presets.ModelBuilder) presets.ObjectComponentFunc {
	return func(obj interface{}, ctx *web.EventContext) h.HTMLComponent {
		gmsgr := presets.MustGetMessages(ctx.R)
		buttonLabel := gmsgr.Create
		m.RightDrawerWidth("800")
		var disableUpdateBtn bool
		isCreateBtn := true
		if ctx.Param(presets.ParamID) != "" {
			isCreateBtn = false
			buttonLabel = gmsgr.Update
			m.RightDrawerWidth("1200")
			disableUpdateBtn = m.Info().Verifier().Do(presets.PermUpdate).ObjectOn(obj).WithReq(ctx.R).IsAllowed() != nil
		}

		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)
		updateBtn := v.VBtn(buttonLabel).
			Color("primary").
			Attr("@click", web.Plaid().
				EventFunc(actions.Update).
				Queries(ctx.Queries()).
				Query(presets.ParamID, ctx.Param(presets.ParamID)).
				URL(m.Info().ListingHref()).
				Go(),
			)
		if disableUpdateBtn {
			updateBtn = updateBtn.Disabled(disableUpdateBtn)
		} else {
			updateBtn = updateBtn.Attr(":disabled", "isFetching").Attr(":loading", "isFetching")
		}
		if isCreateBtn {
			return h.Components(
				v.VSpacer(),
				updateBtn,
			)
		}

		saveNewVersionBtn := v.VBtn(msgr.SaveAsNewVersion).
			Color("secondary").
			Attr("@click", web.Plaid().
				EventFunc(EventSaveNewVersion).
				Queries(ctx.Queries()).
				Query(presets.ParamID, ctx.Param(presets.ParamID)).
				URL(m.Info().ListingHref()).
				Go(),
			)
		if disableUpdateBtn {
			saveNewVersionBtn = saveNewVersionBtn.Disabled(disableUpdateBtn)
		} else {
			saveNewVersionBtn = saveNewVersionBtn.Attr(":disabled", "isFetching").Attr(":loading", "isFetching")
		}

		return h.Components(
			v.VSpacer(),
			saveNewVersionBtn,
			updateBtn,
		)
	}
}

func renameVersionDialog(mb *presets.ModelBuilder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		id := ctx.R.FormValue("rename_id")
		versionName := ctx.R.FormValue("version_name")
		okAction := web.Plaid().
			URL(mb.Info().ListingHref()).
			EventFunc(eventRenameVersionV2).
			Queries(ctx.Queries()).
			Query("rename_id", id).Go()

		r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
			Name: presets.DialogPortalName,
			Body: web.Scope(
				v.VDialog(
					v.VCard(
						v.VCardTitle(h.Text("Version")),
						v.VCardText(
							v.VTextField().Attr(web.VField("VersionName", versionName)...).Variant(v.FieldVariantUnderlined),
						),
						v.VCardActions(
							v.VSpacer(),
							v.VBtn("Cancel").
								Variant(v.VariantFlat).
								Class("ml-2").
								On("click", "locals.renameVersionDialogV2 = false"),

							v.VBtn("OK").
								Color("primary").
								Variant(v.VariantFlat).
								Theme(v.ThemeDark).
								Attr("@click", "locals.renameVersionDialogV2 = false; "+okAction),
						),
					),
				).MaxWidth("420px").Attr("v-model", "locals.renameVersionDialogV2"),
			).Init("{renameVersionDialogV2:true}").VSlot("{locals}"),
		})
		return
	}
}

func renameVersion(mb *presets.ModelBuilder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		paramID := ctx.R.FormValue("rename_id")
		obj := mb.NewModel()
		obj, err = mb.Editing().Fetcher(obj, paramID, ctx)
		if err != nil {
			return
		}

		name := ctx.R.FormValue("VersionName")
		if err = reflectutils.Set(obj, "Version.VersionName", name); err != nil {
			return
		}

		if err = mb.Editing().Saver(obj, paramID, ctx); err != nil {
			return
		}
		qs := ctx.Queries()
		delete(qs, "version_name")
		delete(qs, "rename_id")

		r.RunScript = web.Plaid().URL(ctx.R.RequestURI).Queries(qs).EventFunc(actions.UpdateListingDialog).Go()
		return
	}
}

func deleteVersionDialog(mb *presets.ModelBuilder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		id := ctx.R.FormValue("delete_id")
		versionName := ctx.R.FormValue("version_name")

		utilMsgr := i18n.MustGetModuleMessages(ctx.R, utils.I18nUtilsKey, Messages_en_US).(*utils.Messages)
		// msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)

		r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
			Name: presets.DeleteConfirmPortalName,
			Body: utils.DeleteDialog(
				// TODO i18
				fmt.Sprintf("Are you sure you want to delete %s?", versionName),
				web.Plaid().
					URL(mb.Info().ListingHref()).
					EventFunc(actions.DoDelete).
					Queries(ctx.Queries()).
					Query(presets.ParamInDialog, "true").
					Query(presets.ParamID, id).Go(),
				utilMsgr),
		})
		return
	}
}

func renameVersionAction(_ *gorm.DB, mb *presets.ModelBuilder, _ *Builder, _ *activity.Builder, _ string) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		paramID := ctx.Param(presets.ParamID)

		obj := mb.NewModel()
		obj, err = mb.Editing().Fetcher(obj, paramID, ctx)
		if err != nil {
			return
		}

		name := ctx.R.FormValue("name")

		if err = reflectutils.Set(obj, "Version.VersionName", name); err != nil {
			return
		}

		if err = mb.Editing().Saver(obj, paramID, ctx); err != nil {
			return
		}

		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)
		presets.ShowMessage(&r, msgr.SuccessfullyRename, "")
		return
	}
}
