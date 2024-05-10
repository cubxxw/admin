package pagebuilder

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/qor5/admin/v3/publish"
	"github.com/qor5/admin/v3/seo"
	"github.com/qor5/x/v3/i18n"

	h "github.com/theplant/htmlgo"
	"gorm.io/gorm"

	"github.com/qor5/admin/v3/l10n"
	"github.com/qor5/admin/v3/presets"
	"github.com/qor5/admin/v3/presets/actions"
	. "github.com/qor5/ui/v3/vuetify"
	vx "github.com/qor5/ui/v3/vuetifyx"
	"github.com/qor5/web/v3"
)

func settings(b *Builder) presets.FieldComponentFunc {
	// TODO: refactor versionDialog to use publish/views

	var (
		pm         = b.mb
		seoBuilder = b.seoBuilder
	)
	// publish.ConfigureVersionListDialog(db, b.ps, pm)
	return func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		// TODO: init default VersionComponent

		var (
			start, end, se string
			categories     []*Category
			onlineHint     h.HTMLComponent
		)
		var (
			versionComponent = publish.DefaultVersionComponentFunc(pm)(obj, field, ctx)
			mi               = field.ModelInfo
			p                = obj.(*Page)
			c                = &Category{}
		)

		var (
			previewDevelopUrl = b.previewHref(ctx)
		)

		b.db.First(c, "id = ? AND locale_code = ?", p.CategoryID, p.LocaleCode)
		if p.GetScheduledStartAt() != nil {
			start = p.GetScheduledStartAt().Format("2006-01-02 15:04")
		}
		if p.GetScheduledEndAt() != nil {
			end = p.GetScheduledEndAt().Format("2006-01-02 15:04")
		}
		if start != "" || end != "" {
			se = "Scheduled at: " + start + " ~ " + end
		}

		locale, _ := l10n.IsLocalizableFromCtx(ctx.R.Context())
		seoMsgr := i18n.MustGetModuleMessages(ctx.R, seo.I18nSeoKey, seo.Messages_en_US).(*seo.Messages)
		if err := b.db.Model(&Category{}).Where("locale_code = ?", locale).Find(&categories).Error; err != nil {
			panic(err)
		}

		infoComponentTab := h.Div(
			VTabs(
				VTab(h.Text("Page")).Size(SizeXSmall).Value("Page"),
				VTab(h.Text(seoMsgr.Seo)).Size(SizeXSmall).Value("Seo"),
			).Attr("v-model", "editLocals.infoTab"),
			h.Div(
				VBtn("Save").AppendIcon("mdi-check").Color(ColorSecondary).Size(SizeSmall).Variant(VariantFlat).
					Attr("@click", fmt.Sprintf(`editLocals.infoTab=="Page"?%s:%s`, web.POST().
						EventFunc(actions.Update).
						Query(presets.ParamID, p.PrimarySlug()).
						URL(mi.PresetsPrefix()+"/pages").
						Go(), web.Plaid().
						EventFunc(updateSEOEvent).
						Query(presets.ParamID, p.PrimarySlug()).
						URL(mi.PresetsPrefix()+"/pages").
						Go()),
					),
			),
		).Class("d-flex justify-space-between align-center")

		seoForm := seoBuilder.EditingComponentFunc(obj, nil, ctx)

		infoComponentContent := VTabsWindow(
			VTabsWindowItem(
				b.mb.Editing().ToComponent(pm.Info(), obj, ctx),
			).Value("Page").Class("pt-8"),
			VTabsWindowItem(
				seoForm,
			).Value("Seo").Class("pt-8"),
		).Attr("v-model", "editLocals.infoTab")

		versionBadge := VChip(h.Text(fmt.Sprintf("%d versions", versionCount(b.db, p)))).Color(ColorPrimary).Size(SizeSmall).Class("px-1 mx-1").Attr("style", "height:20px")
		if p.GetStatus() == publish.StatusOnline {
			onlineHint = VAlert(h.Text("The version cannot be edited directly after it is released. Please copy the version and edit it.")).Density(DensityCompact).Type(TypeInfo).Variant(VariantTonal).Closable(true).Class("mb-2")
		}
		return VContainer(
			web.Scope(
				VLayout(
					VCard(
						web.Slot(
							VBtn("").Size(SizeSmall).Icon("mdi-arrow-left").Variant(VariantText).Attr("@click",
								web.GET().URL(mi.PresetsPrefix()+"/pages").PushState(true).Go(),
							),
						).Name(VSlotPrepend),
						web.Slot(h.Text(p.Title),
							versionBadge,
						).Name(VSlotTitle),
						VCardText(
							onlineHint,
							versionComponent,
							h.Div(
								h.Iframe().Src(previewDevelopUrl).Style(`height:320px;width:100%;pointer-events: none;`),
								h.Div(
									h.Div(
										h.Text(se),
									).Class(fmt.Sprintf("bg-%s", ColorSecondaryLighten2)),
									VBtn("Page Builder").PrependIcon("mdi-pencil").Color(ColorSecondary).
										Class("rounded-sm").Height(40).Variant(VariantFlat),
								).Class("pa-6 w-100 d-flex justify-space-between align-center").Style(`position:absolute;top:0;left:0`),
							).Style(`position:relative`).Class("w-100 mt-4").
								Attr("@click",
									web.Plaid().URL(fmt.Sprintf("%s/editors/%v", b.prefix, p.ID)).
										Query(paramPageVersion, p.GetVersion()).
										Query(paramLocale, p.GetLocale()).
										Query(paramPageID, p.PrimarySlug()).
										PushState(true).Go(),
								),
							h.Div(
								h.A(h.Text(previewDevelopUrl)).Href(previewDevelopUrl),
								VBtn("").Icon("mdi-file-document-multiple").Variant(VariantText).Size(SizeXSmall).Class("ml-1").
									Attr("@click", fmt.Sprintf(`$event.view.window.navigator.clipboard.writeText($event.view.window.location.origin+"%s");vars.presetsMessage = { show: true, message: "success", color: "%s"}`, previewDevelopUrl, ColorSuccess)),
							).Class("d-inline-flex align-center"),

							web.Scope(
								infoComponentTab.Class("mt-7"),
								infoComponentContent,
							).VSlot("{form}"),
						),
					).Class("w-75"),
				).Class("d-inline-flex w-100"),
			).VSlot(" { locals : editLocals }").Init(`{ infoTab:"Page"} `),
		)
	}
}

func templateSettings(db *gorm.DB, pm *presets.ModelBuilder) presets.FieldComponentFunc {
	return func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		p := obj.(*Template)

		overview := vx.DetailInfo(
			vx.DetailColumn(
				vx.DetailField(vx.OptionalText(p.Name)).Label("Title"),
				vx.DetailField(vx.OptionalText(p.Description)).Label("Description"),
			),
		)

		editBtn := VBtn("Edit").Variant(VariantFlat).
			Attr("@click", web.POST().
				EventFunc(actions.Edit).
				Query(presets.ParamOverlay, actions.Dialog).
				Query(presets.ParamID, p.PrimarySlug()).
				URL(pm.Info().ListingHref()).Go(),
			)

		return VContainer(
			VRow(
				VCol(
					vx.Card(overview).HeaderTitle("Overview").
						Actions(
							h.If(editBtn != nil, editBtn),
						).Class("mb-4 rounded-lg").Variant(VariantOutlined),
				).Cols(8),
			),
		)
	}
}

func (b *Builder) installDetailingIframeField(templateM *presets.ModelBuilder) {
	b.mb.Detailing().Field("Iframe").ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		var (
			pm         = b.mb
			seoBuilder = b.seoBuilder
		)

		var (
			start, end, se string
			categories     []*Category
			onlineHint     h.HTMLComponent
		)
		var (
			versionComponent = publish.DefaultVersionComponentFunc(pm)(obj, field, ctx)
			mi               = field.ModelInfo
			p                = obj.(*Page)
			c                = &Category{}
		)
		isTemplate := strings.Contains(ctx.R.RequestURI, "/"+templateM.Info().URIName()+"/")
		if v, ok := obj.(interface {
			GetID() uint
		}); ok {
			ctx.R.Form.Set(paramPageID, strconv.Itoa(int(v.GetID())))
		}
		if v, ok := obj.(publish.VersionInterface); ok {
			ctx.R.Form.Set(paramPageVersion, v.GetVersion())
		}
		if l, ok := obj.(l10n.L10nInterface); ok {
			ctx.R.Form.Set(paramLocale, l.GetLocale())
		}
		if isTemplate {
			ctx.R.Form.Set(paramsTpl, "1")
		}

		var (
			previewDevelopUrl = b.previewHref(ctx)
		)

		b.db.First(c, "id = ? AND locale_code = ?", p.CategoryID, p.LocaleCode)
		if p.GetScheduledStartAt() != nil {
			start = p.GetScheduledStartAt().Format("2006-01-02 15:04")
		}
		if p.GetScheduledEndAt() != nil {
			end = p.GetScheduledEndAt().Format("2006-01-02 15:04")
		}
		if start != "" || end != "" {
			se = "Scheduled at: " + start + " ~ " + end
		}

		locale, _ := l10n.IsLocalizableFromCtx(ctx.R.Context())
		seoMsgr := i18n.MustGetModuleMessages(ctx.R, seo.I18nSeoKey, seo.Messages_en_US).(*seo.Messages)
		if err := b.db.Model(&Category{}).Where("locale_code = ?", locale).Find(&categories).Error; err != nil {
			panic(err)
		}

		infoComponentTab := h.Div(
			VTabs(
				VTab(h.Text("Page")).Size(SizeXSmall).Value("Page"),
				VTab(h.Text(seoMsgr.Seo)).Size(SizeXSmall).Value("Seo"),
			).Attr("v-model", "locals.tab"),
			h.Div(
				VBtn("Save").AppendIcon("mdi-check").Color(ColorSecondary).Size(SizeSmall).Variant(VariantFlat).
					Attr("@click", fmt.Sprintf(`editLocals.infoTab=="Page"?%s:%s`, web.POST().
						EventFunc(actions.Update).
						Query(presets.ParamID, p.PrimarySlug()).
						URL(mi.PresetsPrefix()+"/pages").
						Go(), web.Plaid().
						EventFunc(updateSEOEvent).
						Query(presets.ParamID, p.PrimarySlug()).
						URL(mi.PresetsPrefix()+"/pages").
						Go()),
					),
			),
		).Class("d-flex justify-space-between align-center")

		seoForm := seoBuilder.EditingComponentFunc(obj, nil, ctx)

		infoComponentContent := VTabsWindow(
			VTabsWindowItem(
				b.mb.Editing().ToComponent(pm.Info(), obj, ctx),
			).Value("Page").Class("pt-8"),
			VTabsWindowItem(
				seoForm,
			).Value("Seo").Class("pt-8"),
		).Attr("v-model", "locals.tab")

		versionBadge := VChip(h.Text(fmt.Sprintf("%d versions", versionCount(b.db, p)))).Color(ColorPrimary).Size(SizeSmall).Class("px-1 mx-1").Attr("style", "height:20px")
		if p.GetStatus() == publish.StatusOnline {
			onlineHint = VAlert(h.Text("The version cannot be edited directly after it is released. Please copy the version and edit it.")).Density(DensityCompact).Type(TypeInfo).Variant(VariantTonal).Closable(true).Class("mb-2")
		}
		return web.Scope(
			VLayout(
				VAppBar(
					web.Slot(
						VBtn("").Size(SizeXSmall).Icon("mdi-arrow-left").Tile(true).Variant(VariantOutlined).Attr("@click",
							web.GET().URL(mi.PresetsPrefix()+"/pages").PushState(true).Go(),
						),
					).Name(VSlotPrepend),
					web.Slot(
						h.Div(
							h.H1(p.Title),
							versionBadge.Class("mt-2 ml-2"),
						).Class("d-inline-flex align-center"),
					).Name(VSlotTitle),
				).Color(ColorSurface).Elevation(0),
				VMain(
					onlineHint,
					versionComponent,
					h.Div(
						h.Iframe().Src(previewDevelopUrl).Style(`height:320px;width:100%;pointer-events: none;`),
						h.Div(
							h.Div(
								h.Text(se),
							).Class(fmt.Sprintf("bg-%s", ColorSecondaryLighten2)),
							VBtn("Page Builder").PrependIcon("mdi-pencil").Color(ColorSecondary).
								Class("rounded-sm").Height(40).Variant(VariantFlat),
						).Class("pa-6 w-100 d-flex justify-space-between align-center").Style(`position:absolute;top:0;left:0`),
					).Style(`position:relative`).Class("w-100 mt-4").
						Attr("@click",
							web.Plaid().URL(fmt.Sprintf("%s/editors/%v", b.prefix, p.ID)).
								Query(paramPageVersion, p.GetVersion()).
								Query(paramLocale, p.GetLocale()).
								Query(paramPageID, p.PrimarySlug()).
								PushState(true).Go(),
						),
					h.Div(
						h.A(h.Text(previewDevelopUrl)).Href(previewDevelopUrl),
						VBtn("").Icon("mdi-file-document-multiple").Variant(VariantText).Size(SizeXSmall).Class("ml-1").
							Attr("@click", fmt.Sprintf(`$event.view.window.navigator.clipboard.writeText($event.view.window.location.origin+"%s");vars.presetsMessage = { show: true, message: "success", color: "%s"}`, previewDevelopUrl, ColorSuccess)),
					).Class("d-inline-flex align-center"),

					web.Scope(
						infoComponentTab.Class("mt-7"),
						infoComponentContent,
					).VSlot("{form}"),
				).Class("mt-10"),
			),
		).VSlot(" { locals  }").Init(`{ tab:"Page"} `)
	})
}

func (b *Builder) installDetailingEditorField() {
	b.mb.Detailing().Field("Editor").ComponentFunc(func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		return nil
	})
}
