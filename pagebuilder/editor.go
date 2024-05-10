package pagebuilder

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"
	"github.com/sunfmin/reflectutils"
	h "github.com/theplant/htmlgo"
	"goji.io/v3/pat"
	"gorm.io/gorm"

	"github.com/qor5/admin/v3/activity"
	"github.com/qor5/admin/v3/l10n"
	"github.com/qor5/admin/v3/presets"
	"github.com/qor5/admin/v3/presets/actions"
	"github.com/qor5/admin/v3/publish"
	"github.com/qor5/admin/v3/utils"
	. "github.com/qor5/ui/v3/vuetify"
	vx "github.com/qor5/ui/v3/vuetifyx"
	"github.com/qor5/web/v3"
	"github.com/qor5/x/v3/i18n"
)

const (
	AddContainerDialogEvent          = "page_builder_AddContainerDialogEvent"
	AddContainerEvent                = "page_builder_AddContainerEvent"
	DeleteContainerConfirmationEvent = "page_builder_DeleteContainerConfirmationEvent"
	DeleteContainerEvent             = "page_builder_DeleteContainerEvent"
	MoveContainerEvent               = "page_builder_MoveContainerEvent"
	MoveUpDownContainerEvent         = "page_builder_MoveUpDownContainerEvent"
	ToggleContainerVisibilityEvent   = "page_builder_ToggleContainerVisibilityEvent"
	MarkAsSharedContainerEvent       = "page_builder_MarkAsSharedContainerEvent"
	RenameContainerDialogEvent       = "page_builder_RenameContainerDialogEvent"
	RenameContainerEvent             = "page_builder_RenameContainerEvent"
	ShowAddContainerDrawerEvent      = "page_builder_ShowAddContainerDrawerEvent"
	ShowSortedContainerDrawerEvent   = "page_builder_ShowSortedContainerDrawerEvent"
	ShowEditContainerDrawerEvent     = "page_builder_ShowEditContainerDrawerEvent"
	ReloadRenderPageOrTemplateEvent  = "page_builder_ReloadRenderPageOrTemplateEvent"

	paramPageID          = "pageID"
	paramPageVersion     = "pageVersion"
	paramLocale          = "locale"
	paramStatus          = "status"
	paramContainerID     = "containerID"
	paramMoveResult      = "moveResult"
	paramContainerName   = "containerName"
	paramSharedContainer = "sharedContainer"
	paramModelID         = "modelID"
	paramModelName       = "modelName"
	paramMoveDirection   = "paramMoveDirection"
	paramsIsNotEmpty     = "isNotEmpty"
	paramsTpl            = "tpl"
	paramsDevice         = "device"

	DevicePhone    = "phone"
	DeviceTablet   = "tablet"
	DeviceComputer = "computer"

	EventUp     = "up"
	EventDown   = "down"
	EventDelete = "delete"
	EventAdd    = "add"
	EventEdit   = "edit"

	iframeHeightName = "_iframeHeight"

	pageBuilderRightContentPortal = "pageBuilderRightContentPortal"
)

func (b *Builder) Preview(ctx *web.EventContext) (r web.PageResponse, err error) {
	var p *Page
	r.Body, p, err = b.renderPageOrTemplate(ctx, false)
	if err != nil {
		return
	}
	r.PageTitle = p.Title
	return
}

const editorPreviewContentPortal = "editorPreviewContentPortal"

func (b *Builder) Editor(ctx *web.EventContext) (r web.PageResponse, err error) {

	var (
		deviceToggler     h.HTMLComponent
		versionComponent  h.HTMLComponent
		tabContent        web.PageResponse
		activeDevice      int
		pageAppbarContent []h.HTMLComponent
		page              *Page
		exitHref          string

		device = ctx.R.FormValue(paramsDevice)
	)
	ctx.R.Form.Set(presets.ParamID, pat.Param(ctx.R, presets.ParamID))
	switch device {
	case DeviceTablet:
		activeDevice = 1
	case DevicePhone:
		activeDevice = 2
	}
	deviceToggler = web.Scope(
		VBtnToggle(
			VBtn("").Icon("mdi-laptop").Color(ColorPrimary).Variant(VariantText).Class("mr-4").
				Attr("@click", web.Plaid().URL(b.prefix+"/editors").EventFunc(ReloadRenderPageOrTemplateEvent).Queries(ctx.R.Form).Query(paramsDevice, DeviceComputer).Go()),
			VBtn("").Icon("mdi-tablet").Color(ColorPrimary).Variant(VariantText).Class("mr-4").
				Attr("@click", web.Plaid().URL(b.prefix+"/editors").EventFunc(ReloadRenderPageOrTemplateEvent).Queries(ctx.R.Form).Query(paramsDevice, DeviceTablet).Go()),
			VBtn("").Icon("mdi-cellphone").Color(ColorPrimary).Variant(VariantText).Class("mr-4").
				Attr("@click", web.Plaid().URL(b.prefix+"/editors").EventFunc(ReloadRenderPageOrTemplateEvent).Queries(ctx.R.Form).Query(paramsDevice, DevicePhone).Go()),
		).Class("pa-2 rounded-lg ").Attr("v-model", "toggleLocals.activeDevice").Density(DensityCompact),
	).VSlot("{ locals : toggleLocals}").Init(fmt.Sprintf(`{activeDevice: %d}`, activeDevice))

	if tabContent, page, err = b.PageContent(ctx); err != nil {
		return
	}
	if b.mb != nil {
		versionComponent = publish.DefaultVersionComponentFunc(b.mb)(page, &presets.FieldContext{ModelInfo: b.mb.Info()}, ctx)
		exitHref = b.mb.Info().DetailingHref(ctx.R.FormValue(paramPageID))
	}
	pageAppbarContent = h.Components(
		h.Div(
			VIcon("mdi-exit-to-app").Class("mr-4").
				Attr("@click", web.Plaid().URL(exitHref).PushState(true).Go()),
			VToolbarTitle("Page Builder"),
		).Class("d-inline-flex align-center"),
		h.Div(deviceToggler).Class("text-center  w-25 d-flex justify-space-between ml-2"),
		versionComponent,
	)

	r.Body = VApp(
		VAppBar(
			h.Div(
				pageAppbarContent...,
			).Class("d-flex align-center  justify-space-between   border-b w-100").Style("height: 48px"),
		).
			Elevation(0).
			Density("compact").Class("px-6"),
		h.Tag("vx-restore-scroll-listener"),
		vx.VXMessageListener().ListenFunc(b.generateEditorBarJsFunction(ctx)),
		VMain(
			tabContent.Body.(h.HTMLComponent),
		),
	)

	return
}

func (b *Builder) getDevice(ctx *web.EventContext) (device string, style string) {
	device = ctx.R.FormValue(paramsDevice)
	if len(device) == 0 {
		device = b.defaultDevice
	}

	switch device {
	case DevicePhone:
		style = "width: 414px;"
	case DeviceTablet:
		style = "width: 768px;"
		// case Device_Computer:
		//	style = "width: 1264px;"
	}

	return
}

const ContainerToPageLayoutKey = "ContainerToPageLayout"

func (b *Builder) renderPageOrTemplate(ctx *web.EventContext, isEditor bool) (r h.HTMLComponent, p *Page, err error) {
	var (
		isTpl            = ctx.R.FormValue(paramsTpl) != ""
		pageOrTemplateID = ctx.R.FormValue(presets.ParamID)
		version          = ctx.R.FormValue(paramPageVersion)
		locale           = ctx.R.FormValue(paramLocale)
	)

	if isTpl {
		tpl := &Template{}
		err = b.db.First(tpl, "id = ? and locale_code = ?", pageOrTemplateID, locale).Error
		if err != nil {
			return
		}
		p = tpl.Page()
		version = p.Version.Version
	} else {
		err = b.db.First(&p, "id = ? and version = ? and locale_code = ?", pageOrTemplateID, version, locale).Error
		if err != nil {
			return
		}
	}

	var isReadonly bool
	if p.GetStatus() != publish.StatusDraft && isEditor {
		isReadonly = true
	}

	var comps []h.HTMLComponent
	comps, err = b.renderContainers(ctx, p, isEditor, isReadonly)
	if err != nil {
		return
	}
	r = h.Components(comps...)
	if b.pageLayoutFunc != nil {
		var seoTags h.HTMLComponent
		if b.seoBuilder != nil {
			seoTags = b.seoBuilder.Render(p, ctx.R)
		}
		input := &PageLayoutInput{
			IsEditor:  isEditor,
			IsPreview: !isEditor,
			Page:      p,
			SeoTags:   seoTags,
		}

		if isEditor {
			input.EditorCss = append(input.EditorCss, h.RawHTML(`<link rel="stylesheet" href="https://fonts.googleapis.com/icon?family=Material+Icons">`))
			input.EditorCss = append(input.EditorCss, h.Style(`
			.inner-shadow {
			  position: absolute;
			  width: 100%;
			  height: 100%;
			  opacity: 0;
			  top: 0;
			  left: 0;
			  box-shadow: 3px 3px 0 0px #3E63DD inset, -3px 3px 0 0px #3E63DD inset;
			}
			
			.editor-add {
			  width: 100%;
			  position: absolute;
			  z-index: 9998;
			  opacity: 0;
			  transition: opacity .5s ease-in-out;
			  text-align: center;
			}
			
			.editor-add div {
			  width: 100%;
			  background-color: #3E63DD;
			  transition: height .5s ease-in-out;
			  height: 3px;
			}
			
			.editor-add button {
			  color: #FFFFFF;
			  background-color: #3E63DD;
			  pointer-event: none;
			}
			
			.wrapper-shadow:hover {
			  cursor: pointer;
			}
			
			.wrapper-shadow:hover .editor-add {
			  opacity: 1;
			}
			
			.wrapper-shadow:hover .editor-add div {
			  height: 6px;
			}
			
			.editor-bar {
			  position: absolute;
			  z-index: 9999;
			  width: 30%;
			  opacity: 0;
			  display: flex;
			  background-color: #3E63DD;
			  justify-content: space-between;
			}
			
			.editor-bar button {
			  color: #FFFFFF;
			  background-color: #3E63DD;
			}
			
			.editor-bar h6 {
			  color: #FFFFFF;
			}
			
			.highlight .editor-bar {
			  opacity: 1;
			}
			
			.highlight .editor-add {
			  opacity: 1;
			}
			
			.highlight .inner-shadow {
			  opacity: 1;
			}
`))
		}
		if f := ctx.R.Context().Value(ContainerToPageLayoutKey); f != nil {
			pl, ok := f.(*PageLayoutInput)
			if ok {
				input.FreeStyleCss = append(input.FreeStyleCss, pl.FreeStyleCss...)
				input.FreeStyleTopJs = append(input.FreeStyleTopJs, pl.FreeStyleTopJs...)
				input.FreeStyleBottomJs = append(input.FreeStyleBottomJs, pl.FreeStyleBottomJs...)
				input.Hreflang = pl.Hreflang
			}
		}

		if isEditor {
			// use newCtx to avoid inserting page head to head outside of iframe
			newCtx := &web.EventContext{
				R:        ctx.R,
				Injector: &web.PageInjector{},
			}
			r = b.pageLayoutFunc(h.Components(comps...), input, newCtx)
			newCtx.Injector.HeadHTMLComponent("style", b.pageStyle, true)
			r = h.HTMLComponents{
				h.RawHTML("<!DOCTYPE html>\n"),
				h.Tag("html").Children(
					h.Head(
						newCtx.Injector.GetHeadHTMLComponent(),
					),
					h.Body(
						h.Div(
							r,
						).Id("app").Attr("v-cloak", true),
						newCtx.Injector.GetTailHTMLComponent(),
					).Class("front"),
				).AttrIf("lang", newCtx.Injector.GetHTMLLang(), newCtx.Injector.GetHTMLLang() != ""),
			}
			iframeHeightCookie, _ := ctx.R.Cookie(iframeHeightName)
			_, width := b.getDevice(ctx)
			iframeValue := "1000px"
			if iframeHeightCookie != nil {
				iframeValue = iframeHeightCookie.Value
			}
			r = h.Div(
				h.Tag("vx-scroll-iframe").Attr(
					":srcdoc", h.JSONString(h.MustString(r, ctx.R.Context()))).
					Attr(":iframe-height-name", h.JSONString(iframeHeightName)).
					Attr(":iframe-value", h.JSONString(iframeValue)).
					Attr("ref", "scrollIframe"),
			).Class("page-builder-container mx-auto").Attr("style", width)

		} else {
			r = b.pageLayoutFunc(h.Components(comps...), input, ctx)
			ctx.Injector.HeadHTMLComponent("style", b.pageStyle, true)
		}
	}

	return
}

func (b *Builder) renderContainers(ctx *web.EventContext, p *Page, isEditor bool, isReadonly bool) (r []h.HTMLComponent, err error) {
	var cons []*Container
	err = b.db.Order("display_order ASC").Find(&cons, "page_id = ? AND page_version = ? AND locale_code = ?", p.ID, p.GetVersion(), p.GetLocale()).Error
	if err != nil {
		return
	}
	device, _ := b.getDevice(ctx)
	cbs := b.getContainerBuilders(cons)
	for i, ec := range cbs {
		if ec.container.Hidden {
			continue
		}
		obj := ec.builder.NewModel()
		err = b.db.FirstOrCreate(obj, "id = ?", ec.container.ModelID).Error
		if err != nil {
			return
		}
		var displayName = i18n.T(ctx.R, presets.ModelsI18nModuleKey, ec.container.DisplayName)
		input := RenderInput{
			Page:        p,
			IsEditor:    isEditor,
			IsReadonly:  isReadonly,
			Device:      device,
			ContainerId: ec.container.PrimarySlug(),
			DisplayName: displayName,
			IsFirst:     i == 0,
			IsEnd:       i == len(cbs)-1,
			HighLight:   ctx.R.FormValue(paramModelID) == strconv.Itoa(int(ec.container.ModelID)),
			ModelName:   ec.container.ModelName,
		}
		pure := ec.builder.renderFunc(obj, &input, ctx)
		r = append(r, pure)
	}

	return
}

type ContainerSorterItem struct {
	Index          int    `json:"index"`
	Label          string `json:"label"`
	ModelName      string `json:"model_name"`
	ModelID        string `json:"model_id"`
	DisplayName    string `json:"display_name"`
	ContainerID    string `json:"container_id"`
	URL            string `json:"url"`
	Shared         bool   `json:"shared"`
	Hidden         bool   `json:"hidden"`
	VisibilityIcon string `json:"visibility_icon"`
	ParamID        string `json:"param_id"`
	Locale         string `json:"locale"`
}

type ContainerSorter struct {
	Items []ContainerSorterItem `json:"items"`
}

func (b *Builder) renderContainersList(ctx *web.EventContext) (r h.HTMLComponent) {
	r = VLayout(
		VMain(
			b.ContainerComponent(ctx),
		),
	)
	return
}
func (b *Builder) renderEditContainer(ctx *web.EventContext) (r h.HTMLComponent, err error) {

	var (
		pageID        = ctx.R.FormValue(presets.ParamID)
		modelName     = ctx.R.FormValue(paramModelName)
		containerName = ctx.R.FormValue(paramContainerName)
		pageVersion   = ctx.R.FormValue(paramPageVersion)
		locale        = ctx.R.FormValue(paramLocale)
		modelID       = ctx.R.FormValue(paramModelID)
	)
	builder := b.ContainerByName(modelName).GetModelBuilder()
	element := builder.NewModel()
	if err = b.db.First(element, modelID).Error; err != nil {
		return
	}
	r = web.Scope(
		VLayout(
			VMain(
				h.Div(
					h.Div(VBtn("").Size(SizeSmall).Icon("mdi-arrow-left").Variant(VariantText).
						Attr("@click", web.Plaid().
							URL(fmt.Sprintf("%s/editors/%s?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
							EventFunc(ShowAddContainerDrawerEvent).
							Queries(ctx.R.Form).
							Go(),
						),
						h.Span(containerName).Class("text-subtitle-1"),
					),
					h.Div(
						VBtn("Cancel").Variant(VariantOutlined).Size(SizeSmall).Class("mr-2"),
						VBtn("Save").Variant(VariantFlat).Color("secondary").Size(SizeSmall).Attr("@click", web.Plaid().
							EventFunc(actions.Update).
							URL(ctx.R.URL.Path).
							Query(presets.ParamID, modelID).
							Go()),
					),
				).Class("d-flex  pa-6 align-center justify-space-between"),
				VDivider(),
				h.Div(
					builder.Editing().ToComponent(builder.Info(), element, ctx),
				).Class("pa-6"),
			),
		),
	).VSlot("{ form }")
	return
}
func (b *Builder) renderContainersSortedList(ctx *web.EventContext) (r h.HTMLComponent, err error) {
	var (
		cons         []*Container
		pageID       = ctx.R.FormValue(presets.ParamID)
		pageVersion  = ctx.R.FormValue(paramPageVersion)
		locale       = ctx.R.FormValue(paramLocale)
		status       = ctx.R.FormValue(paramStatus)
		isReadonly   = status != publish.StatusDraft
		msgr         = i18n.MustGetModuleMessages(ctx.R, I18nPageBuilderKey, Messages_en_US).(*Messages)
		activityMsgr = i18n.MustGetModuleMessages(ctx.R, activity.I18nActivityKey, activity.Messages_en_US).(*activity.Messages)
	)

	err = b.db.Order("display_order ASC").Find(&cons, "page_id = ? AND page_version = ? AND locale_code = ?", pageID, pageVersion, locale).Error
	if err != nil {
		return
	}

	var sorterData ContainerSorter
	sorterData.Items = []ContainerSorterItem{}
	if len(cons) > 0 {
		ctx.R.Form.Set(paramsIsNotEmpty, "1")
	}
	for i, c := range cons {
		vicon := "mdi-eye"
		if c.Hidden {
			vicon = "mdi-eye-off"
		}
		var displayName = i18n.T(ctx.R, presets.ModelsI18nModuleKey, c.DisplayName)

		sorterData.Items = append(sorterData.Items,
			ContainerSorterItem{
				Index:          i,
				Label:          inflection.Plural(strcase.ToKebab(c.ModelName)),
				ModelName:      c.ModelName,
				ModelID:        strconv.Itoa(int(c.ModelID)),
				DisplayName:    displayName,
				ContainerID:    strconv.Itoa(int(c.ID)),
				URL:            b.ContainerByName(c.ModelName).mb.Info().ListingHref(),
				Shared:         c.Shared,
				VisibilityIcon: vicon,
				ParamID:        c.PrimarySlug(),
				Locale:         locale,
				Hidden:         c.Hidden,
			},
		)
	}
	var (
		menu = VMenu(
			web.Slot(
				VBtn("").Icon("mdi-dots-horizontal").Variant(VariantText).Size(SizeSmall).Attr("v-bind", "props").Attr("v-show", "element.editShow || (isActive || isHovering)"),
			).Name("activator").Scope("{isActive,props}"),
			VList(
				VListItem(
					VBtn(msgr.Rename).PrependIcon("mdi-pencil").Attr("@click",
						"element.editShow=!element.editShow",
					),
				),
				VListItem(
					VBtn(activityMsgr.ActionDelete).PrependIcon("mdi-delete").Attr("@click",
						web.Plaid().
							URL(web.Var("element.url")).
							EventFunc(DeleteContainerConfirmationEvent).
							Query(paramContainerID, web.Var("element.param_id")).
							Query(paramContainerName, web.Var("element.display_name")).
							Go(),
					),
				),
			),
		)
	)

	r = web.Scope(
		VSheet(
			h.Div(
				h.Div(h.Span("Elements").Class("text-subtitle-1")),
				h.If(!isReadonly,
					VBtn("ADD").Size(SizeSmall).Variant(VariantFlat).Color("primary").
						Attr("@click",
							web.Plaid().
								URL(fmt.Sprintf("%s/editors/%s?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
								EventFunc(ShowAddContainerDrawerEvent).
								Queries(ctx.R.Form).
								Form(nil).
								Go(),
						),
				),
			).Class("d-flex justify-space-between pa-6 align-center "),

			VList(
				h.Tag("vx-draggable").
					Attr("item-key", "model_id").
					Attr("v-model", "sortLocals.items", "handle", ".handle", "animation", "300").
					Attr("@end", web.Plaid().
						URL(fmt.Sprintf("%s/editors", b.prefix)).
						EventFunc(MoveContainerEvent).
						Queries(ctx.R.Form).
						FieldValue(paramMoveResult, web.Var("JSON.stringify(sortLocals.items)")).
						Go()).Children(
					h.Template(
						h.Div(
							VHover(
								web.Slot(
									VListItem(
										web.Slot(
											h.If(!isReadonly,
												VBtn("").Variant(VariantText).Icon("mdi-drag").Class("my-2 ml-1 mr-1").Attr(":class", `element.hidden?"":"handle"`),
											),
										).Name("prepend"),
										VListItemTitle(
											VListItem(
												web.Scope(
													VTextField().HideDetails(true).Density(DensityCompact).Color(ColorPrimary).Autofocus(true).Variant(FieldVariantOutlined).
														Attr("v-model", "form.DisplayName").
														Attr("v-if", "element.editShow").
														Attr("@blur", "element.editShow=false").
														Attr("@keyup.enter", web.Plaid().
															URL(fmt.Sprintf("%s/editors", b.prefix)).
															EventFunc(RenameContainerEvent).Query(paramContainerID, web.Var("element.param_id")).Go()),
													VListItemTitle(h.Text("{{element.display_name}}")).Attr(":style", "[element.shared ? {'color':'green'}:{}]").Attr("v-if", "!element.editShow"),
												).VSlot("{form}").FormInit("{ DisplayName:element.display_name }"),
											),
										),
										web.Slot(
											h.If(!isReadonly,
												h.Div(
													VBtn("").Variant(VariantText).Attr(":icon", "element.visibility_icon").Size(SizeSmall).Attr("@click",
														web.Plaid().
															URL(web.Var("element.url")).
															EventFunc(ToggleContainerVisibilityEvent).
															Queries(ctx.R.Form).
															Query(paramContainerID, web.Var("element.param_id")).
															Go(),
													).Attr("v-show", "element.editShow || (element.hidden || isHovering)"),

													VBtn("").Variant(VariantText).Icon("mdi-cog").Size(SizeSmall).Attr("@click",
														web.Plaid().
															URL(web.Var("element.url")).
															EventFunc(ShowEditContainerDrawerEvent).
															Queries(ctx.R.Form).
															Query(paramModelID, web.Var("element.model_id")).
															Query(paramContainerName, web.Var("element.display_name")).
															Query(paramModelName, web.Var("element.model_name")).
															Go(),
													).Attr("v-show", "element.editShow || isHovering"),
													menu,
												),
											),
										).Name("append"),
									).Attr(":variant", fmt.Sprintf(` element.hidden &&!isHovering && !element.editShow?"%s":"%s"`, VariantPlain, VariantText)).
										Attr("v-bind", "props").
										Attr("@click", fmt.Sprintf(`locals.el.refs.scrollIframe.scrollToCurrentContainer(%s+"_"+%s);`, web.Var("element.label"), web.Var("element.model_id"))),
								).Name("default").Scope("{ isHovering, props }"),
							),
							VDivider(),
						),
					).Attr("#item", " { element } "),
				),
			),
		).Class("pa-4 pt-2"),
	).Init(h.JSONString(sorterData)).VSlot("{ locals:sortLocals,form }")
	return
}

func (b *Builder) AddContainer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var (
		pageID          = ctx.QueryAsInt(presets.ParamID)
		pageVersion     = ctx.R.FormValue(paramPageVersion)
		locale          = ctx.R.FormValue(paramLocale)
		containerName   = ctx.R.FormValue(paramContainerName)
		sharedContainer = ctx.R.FormValue(paramSharedContainer)
		modelID         = ctx.QueryAsInt(paramModelID)
		containerID     = ctx.R.FormValue(paramContainerID)
	)
	if sharedContainer == "true" {
		err = b.AddSharedContainerToPage(pageID, containerID, pageVersion, locale, containerName, uint(modelID))
	} else {
		var newModelId uint
		newModelId, err = b.AddContainerToPage(pageID, containerID, pageVersion, locale, containerName)
		modelID = int(newModelId)
	}
	r.RunScript = web.Plaid().
		URL(b.ContainerByName(containerName).mb.Info().ListingHref()).
		EventFunc(ReloadRenderPageOrTemplateEvent).
		Queries(ctx.R.Form).
		Query(paramModelID, modelID).
		Go() + ";" + web.Plaid().
		URL(b.ContainerByName(containerName).mb.Info().ListingHref()).
		EventFunc(ShowEditContainerDrawerEvent).
		Queries(ctx.R.Form).
		Query(paramModelID, modelID).
		Go()
	return
}

func (b *Builder) MoveContainer(ctx *web.EventContext) (r web.EventResponse, err error) {
	moveResult := ctx.R.FormValue(paramMoveResult)

	var result []ContainerSorterItem
	err = json.Unmarshal([]byte(moveResult), &result)
	if err != nil {
		return
	}
	err = b.db.Transaction(func(tx *gorm.DB) (inerr error) {
		for i, r := range result {
			if inerr = tx.Model(&Container{}).Where("id = ? AND locale_code = ?", r.ContainerID, r.Locale).Update("display_order", i+1).Error; inerr != nil {
				return
			}
		}
		return
	})
	ctx.R.Form.Del(paramMoveResult)
	r.RunScript = web.Plaid().EventFunc(ReloadRenderPageOrTemplateEvent).URL(ctx.R.URL.Path).Form(nil).Queries(ctx.R.Form).Go()
	return
}

func (b *Builder) MoveUpDownContainer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var (
		container    Container
		preContainer Container
	)
	paramID := ctx.R.FormValue(paramContainerID)
	direction := ctx.R.FormValue(paramMoveDirection)
	cs := container.PrimaryColumnValuesBySlug(paramID)
	containerID := cs["id"]
	locale := cs["locale_code"]

	err = b.db.Transaction(func(tx *gorm.DB) (inerr error) {
		if inerr = tx.Where("id = ? AND locale_code = ?", containerID, locale).First(&container).Error; inerr != nil {
			return
		}
		g := tx.Model(&Container{}).Where("page_id = ? AND page_version = ? AND locale_code = ? ", container.PageID, container.PageVersion, container.LocaleCode)
		if direction == EventUp {
			g = g.Where("display_order < ? ", container.DisplayOrder).Order(" display_order desc ")
		} else {
			g = g.Where("display_order > ? ", container.DisplayOrder).Order(" display_order asc ")
		}
		g.First(&preContainer)
		if preContainer.ID <= 0 {
			return
		}
		if inerr = tx.Model(&Container{}).Where("id = ? AND locale_code = ?", containerID, locale).Update("display_order", preContainer.DisplayOrder).Error; inerr != nil {
			return
		}
		if inerr = tx.Model(&Container{}).Where("id = ? AND locale_code = ?", preContainer.ID, preContainer.LocaleCode).Update("display_order", container.DisplayOrder).Error; inerr != nil {
			return
		}
		return
	})
	r.RunScript = web.Plaid().EventFunc(ReloadRenderPageOrTemplateEvent).URL(ctx.R.URL.Path).Queries(ctx.R.Form).Go()
	return
}

func (b *Builder) ToggleContainerVisibility(ctx *web.EventContext) (r web.EventResponse, err error) {
	var container Container
	paramID := ctx.R.FormValue(paramContainerID)
	pageID := ctx.R.FormValue(paramPageID)
	pageVersion := ctx.R.FormValue(paramPageVersion)
	cs := container.PrimaryColumnValuesBySlug(paramID)
	containerID := cs["id"]
	locale := cs["locale_code"]

	err = b.db.Exec("UPDATE page_builder_containers SET hidden = NOT(coalesce(hidden,FALSE)) WHERE id = ? AND locale_code = ?", containerID, locale).Error
	r.RunScript = web.Plaid().EventFunc(ReloadRenderPageOrTemplateEvent).URL(ctx.R.URL.Path).Queries(ctx.R.Form).Go() +
		";" +
		web.Plaid().
			URL(fmt.Sprintf("%s/editors/%s?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
			EventFunc(ShowSortedContainerDrawerEvent).
			Queries(ctx.R.Form).
			Go()
	return
}

func (b *Builder) DeleteContainerConfirmation(ctx *web.EventContext) (r web.EventResponse, err error) {
	paramID := ctx.R.FormValue(paramContainerID)

	containerName := ctx.R.FormValue(paramContainerName)

	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
		Name: presets.DeleteConfirmPortalName,
		Body: web.Scope(
			VDialog(
				VCard(
					VCardTitle(h.Text(fmt.Sprintf("Are you sure you want to delete %s?", containerName))),
					VCardActions(
						VSpacer(),
						VBtn("Cancel").
							Variant(VariantFlat).
							Class("ml-2").
							Attr("@click", "dialogLocals.deleteConfirmation = false"),

						VBtn("Delete").
							Color("primary").
							Variant(VariantFlat).
							Theme(ThemeDark).
							Attr("@click", web.Plaid().
								URL(fmt.Sprintf("%s/editors", b.prefix)).
								EventFunc(DeleteContainerEvent).
								Query(paramContainerID, paramID).
								Go()),
					),
				),
			).MaxWidth("600px").
				Attr("v-model", "dialogLocals.deleteConfirmation"),
		).VSlot(`{ locals : dialogLocals }`).Init(`{deleteConfirmation: true}`),
	})

	return
}

func (b *Builder) DeleteContainer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var container Container
	paramID := ctx.R.FormValue(paramContainerID)
	cs := container.PrimaryColumnValuesBySlug(paramID)
	containerID := cs["id"]
	locale := cs["locale_code"]

	err = b.db.Delete(&Container{}, "id = ? AND locale_code = ?", containerID, locale).Error
	if err != nil {
		return
	}
	r.PushState = web.Location(url.Values{})
	return
}

func (b *Builder) AddContainerToPage(pageID int, containerID, pageVersion, locale, containerName string) (modelID uint, err error) {
	model := b.ContainerByName(containerName).NewModel()
	var dc DemoContainer
	b.db.Where("model_name = ? AND locale_code = ?", containerName, locale).First(&dc)
	if dc.ID != 0 && dc.ModelID != 0 {
		b.db.Where("id = ?", dc.ModelID).First(model)
		reflectutils.Set(model, "ID", uint(0))
	}

	err = b.db.Create(model).Error
	if err != nil {
		return
	}

	var (
		maxOrder     sql.NullFloat64
		displayOrder float64
	)
	err = b.db.Model(&Container{}).Select("MAX(display_order)").Where("page_id = ? and page_version = ? and locale_code = ?", pageID, pageVersion, locale).Scan(&maxOrder).Error
	if err != nil {
		return
	}
	if containerID != "" {
		var lastContainer Container
		cs := lastContainer.PrimaryColumnValuesBySlug(containerID)
		if err = b.db.Where("id = ? AND locale_code = ?", cs["id"], locale).First(&lastContainer).Error; err != nil {
			return
		}
		displayOrder = lastContainer.DisplayOrder
		if err = b.db.Model(&Container{}).Where("page_id = ? and page_version = ? and locale_code = ? and display_order > ? ", pageID, pageVersion, locale, displayOrder).
			UpdateColumn("display_order", gorm.Expr("display_order + ? ", 1)).Error; err != nil {
			return
		}
	} else {
		displayOrder = maxOrder.Float64
	}
	modelID = reflectutils.MustGet(model, "ID").(uint)
	err = b.db.Create(&Container{
		PageID:       uint(pageID),
		PageVersion:  pageVersion,
		ModelName:    containerName,
		DisplayName:  containerName,
		ModelID:      modelID,
		DisplayOrder: displayOrder + 1,
		Locale: l10n.Locale{
			LocaleCode: locale,
		},
	}).Error
	return
}

func (b *Builder) AddSharedContainerToPage(pageID int, containerID, pageVersion, locale, containerName string, modelID uint) (err error) {
	var c Container
	err = b.db.First(&c, "model_name = ? AND model_id = ? AND shared = true", containerName, modelID).Error
	if err != nil {
		return
	}
	var (
		maxOrder     sql.NullFloat64
		displayOrder float64
	)
	err = b.db.Model(&Container{}).Select("MAX(display_order)").Where("page_id = ? and page_version = ? and locale_code = ?", pageID, pageVersion, locale).Scan(&maxOrder).Error
	if err != nil {
		return
	}
	if containerID != "" {
		var lastContainer Container
		cs := lastContainer.PrimaryColumnValuesBySlug(containerID)
		if err = b.db.Where("id = ? AND locale_code = ?", cs["id"], locale).First(&lastContainer).Error; err != nil {
			return
		}
		displayOrder = lastContainer.DisplayOrder
		if err = b.db.Model(&Container{}).Where("page_id = ? and page_version = ? and locale_code = ? and display_order > ? ", pageID, pageVersion, locale, displayOrder).
			UpdateColumn("display_order", gorm.Expr("display_order + ? ", 1)).Error; err != nil {
			return
		}
	} else {
		displayOrder = maxOrder.Float64
	}
	err = b.db.Create(&Container{
		PageID:       uint(pageID),
		PageVersion:  pageVersion,
		ModelName:    containerName,
		DisplayName:  c.DisplayName,
		ModelID:      modelID,
		Shared:       true,
		DisplayOrder: displayOrder + 1,
		Locale: l10n.Locale{
			LocaleCode: locale,
		},
	}).Error
	if err != nil {
		return
	}
	return
}

func (b *Builder) copyContainersToNewPageVersion(db *gorm.DB, pageID int, locale, oldPageVersion, newPageVersion string) (err error) {
	return b.copyContainersToAnotherPage(db, pageID, oldPageVersion, locale, pageID, newPageVersion, locale)
}

func (b *Builder) copyContainersToAnotherPage(db *gorm.DB, pageID int, pageVersion, locale string, toPageID int, toPageVersion, toPageLocale string) (err error) {
	var cons []*Container
	err = db.Order("display_order ASC").Find(&cons, "page_id = ? AND page_version = ? AND locale_code = ?", pageID, pageVersion, locale).Error
	if err != nil {
		return
	}

	for _, c := range cons {
		newModelID := c.ModelID
		if !c.Shared {
			model := b.ContainerByName(c.ModelName).NewModel()
			if err = db.First(model, "id = ?", c.ModelID).Error; err != nil {
				return
			}
			if err = reflectutils.Set(model, "ID", uint(0)); err != nil {
				return
			}
			if err = db.Create(model).Error; err != nil {
				return
			}
			newModelID = reflectutils.MustGet(model, "ID").(uint)
		}

		if err = db.Create(&Container{
			PageID:       uint(toPageID),
			PageVersion:  toPageVersion,
			ModelName:    c.ModelName,
			DisplayName:  c.DisplayName,
			ModelID:      newModelID,
			DisplayOrder: c.DisplayOrder,
			Shared:       c.Shared,
			Locale: l10n.Locale{
				LocaleCode: toPageLocale,
			},
		}).Error; err != nil {
			return
		}
	}
	return
}

func (b *Builder) localizeContainersToAnotherPage(db *gorm.DB, pageID int, pageVersion, locale string, toPageID int, toPageVersion, toPageLocale string) (err error) {
	var cons []*Container
	err = db.Order("display_order ASC").Find(&cons, "page_id = ? AND page_version = ? AND locale_code = ?", pageID, pageVersion, locale).Error
	if err != nil {
		return
	}

	for _, c := range cons {
		newModelID := c.ModelID
		newDisplayName := c.DisplayName
		if !c.Shared {
			model := b.ContainerByName(c.ModelName).NewModel()
			if err = db.First(model, "id = ?", c.ModelID).Error; err != nil {
				return
			}
			if err = reflectutils.Set(model, "ID", uint(0)); err != nil {
				return
			}
			if err = db.Create(model).Error; err != nil {
				return
			}
			newModelID = reflectutils.MustGet(model, "ID").(uint)
		} else {
			var count int64
			var sharedCon Container
			if err = db.Where("model_name = ? AND localize_from_model_id = ? AND locale_code = ? AND shared = ?", c.ModelName, c.ModelID, toPageLocale, true).First(&sharedCon).Count(&count).Error; err != nil && err != gorm.ErrRecordNotFound {
				return
			}

			if count == 0 {
				model := b.ContainerByName(c.ModelName).NewModel()
				if err = db.First(model, "id = ?", c.ModelID).Error; err != nil {
					return
				}
				if err = reflectutils.Set(model, "ID", uint(0)); err != nil {
					return
				}
				if err = db.Create(model).Error; err != nil {
					return
				}
				newModelID = reflectutils.MustGet(model, "ID").(uint)
			} else {
				newModelID = sharedCon.ModelID
				newDisplayName = sharedCon.DisplayName
			}
		}

		var newCon Container
		err = db.Order("display_order ASC").Find(&newCon, "id = ? AND locale_code = ?", c.ID, toPageLocale).Error
		if err != nil {
			return
		}

		newCon.ID = c.ID
		newCon.PageID = uint(toPageID)
		newCon.PageVersion = toPageVersion
		newCon.ModelName = c.ModelName
		newCon.DisplayName = newDisplayName
		newCon.ModelID = newModelID
		newCon.DisplayOrder = c.DisplayOrder
		newCon.Shared = c.Shared
		newCon.LocaleCode = toPageLocale
		newCon.LocalizeFromModelID = c.ModelID

		if err = db.Save(&newCon).Error; err != nil {
			return
		}
	}
	return
}

func (b *Builder) localizeCategory(db *gorm.DB, fromCategoryID uint, fromLocale string, toLocale string) (err error) {
	if fromCategoryID == 0 {
		return
	}
	var category Category
	var toCategory Category
	err = db.First(&category, "id = ? AND locale_code = ?", fromCategoryID, fromLocale).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
		return
	}
	if err != nil {
		return
	}
	err = db.First(&toCategory, "id = ? AND locale_code = ?", fromCategoryID, toLocale).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		category.LocaleCode = toLocale
		err = db.Save(&category).Error
		return
	}
	return
}

func (b *Builder) createModelAfterLocalizeDemoContainer(db *gorm.DB, c *DemoContainer) (err error) {
	model := b.ContainerByName(c.ModelName).NewModel()
	if err = db.First(model, "id = ?", c.ModelID).Error; err != nil {
		return
	}
	if err = reflectutils.Set(model, "ID", uint(0)); err != nil {
		return
	}
	if err = db.Create(model).Error; err != nil {
		return
	}

	c.ModelID = reflectutils.MustGet(model, "ID").(uint)
	return
}

func (b *Builder) MarkAsSharedContainer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var container Container
	paramID := ctx.R.FormValue(paramContainerID)
	cs := container.PrimaryColumnValuesBySlug(paramID)
	containerID := cs["id"]
	locale := cs["locale_code"]

	err = b.db.Model(&Container{}).Where("id = ? AND locale_code = ?", containerID, locale).Update("shared", true).Error
	if err != nil {
		return
	}
	r.PushState = web.Location(url.Values{})
	return
}

func (b *Builder) RenameContainer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var container Container
	paramID := ctx.R.FormValue(paramContainerID)
	cs := container.PrimaryColumnValuesBySlug(paramID)
	containerID := cs["id"]
	locale := cs["locale_code"]
	name := ctx.R.FormValue("DisplayName")
	var c Container
	err = b.db.First(&c, "id = ? AND locale_code = ?  ", containerID, locale).Error
	if err != nil {
		return
	}
	if c.Shared {
		err = b.db.Model(&Container{}).Where("model_name = ? AND model_id = ? AND locale_code = ?", c.ModelName, c.ModelID, locale).Update("display_name", name).Error
		if err != nil {
			return
		}
	} else {
		err = b.db.Model(&Container{}).Where("id = ? AND locale_code = ?", containerID, locale).Update("display_name", name).Error
		if err != nil {
			return
		}
	}

	r.PushState = web.Location(url.Values{})
	return
}

func (b *Builder) RenameContainerDialog(ctx *web.EventContext) (r web.EventResponse, err error) {
	paramID := ctx.R.FormValue(paramContainerID)
	name := ctx.R.FormValue(paramContainerName)
	okAction := web.Plaid().
		URL(fmt.Sprintf("%s/editors", b.prefix)).
		EventFunc(RenameContainerEvent).Query(paramContainerID, paramID).Go()
	portalName := dialogPortalName
	if ctx.R.FormValue("portal") == "presets" {
		portalName = presets.DialogPortalName
	}
	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
		Name: portalName,
		Body: web.Scope(
			VDialog(
				VCard(
					VCardTitle(h.Text("Rename")),
					VCardText(
						VTextField().Attr(web.VField("DisplayName", name)...).Variant(FieldVariantUnderlined),
					),
					VCardActions(
						VSpacer(),
						VBtn("Cancel").
							Variant(VariantFlat).
							Class("ml-2").
							On("click", "locals.renameDialog = false"),

						VBtn("OK").
							Color("primary").
							Variant(VariantFlat).
							Theme(ThemeDark).
							Attr("@click", okAction),
					),
				),
			).MaxWidth("400px").
				Attr("v-model", "locals.renameDialog"),
		).Init("{renameDialog:true}").VSlot("{locals}"),
	})
	return
}

func (b *Builder) ContainerComponent(ctx *web.EventContext) (component h.HTMLComponent) {
	var (
		pageID      = ctx.R.FormValue(presets.ParamID)
		pageVersion = ctx.R.FormValue(paramPageVersion)
		locale      = ctx.R.FormValue(paramLocale)
		containerId = ctx.R.FormValue(paramContainerID)
		msgr        = i18n.MustGetModuleMessages(ctx.R, I18nPageBuilderKey, Messages_en_US).(*Messages)
	)
	var (
		containers  []h.HTMLComponent
		groupsNames []string
	)
	sort.Slice(b.containerBuilders, func(i, j int) bool {
		return b.containerBuilders[i].group != "" && b.containerBuilders[j].group == ""
	})
	var groupContainers = utils.GroupBySlice[*ContainerBuilder, string](b.containerBuilders, func(builder *ContainerBuilder) string {
		return builder.group
	})
	for _, group := range groupContainers {
		if len(group) == 0 {
			break
		}
		var groupName = group[0].group
		if groupName == "" {
			groupName = "Others"
		}
		if b.expendContainers {
			groupsNames = append(groupsNames, groupName)
		}
		var listItems []h.HTMLComponent
		for _, builder := range group {
			cover := builder.cover
			if cover == "" {
				cover = path.Join(b.prefix, b.imagesPrefix, strings.ReplaceAll(builder.name, " ", "")+".png")
			}
			containerName := i18n.T(ctx.R, presets.ModelsI18nModuleKey, builder.name)
			listItems = append(listItems, VListItem(
				VListItemTitle(h.Text(containerName)),
				VListItemSubtitle(VImg().Src(cover).Height(100)),
			).Attr("@click",
				web.Plaid().
					URL(fmt.Sprintf("%s/editors/%s?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
					EventFunc(AddContainerEvent).
					Query(presets.ParamID, ctx.R.FormValue(presets.ParamID)).
					Query(paramStatus, ctx.R.FormValue(paramStatus)).
					Query(paramModelName, builder.name).
					Query(paramPageVersion, pageVersion).
					Query(paramLocale, locale).
					Query(paramContainerName, builder.name).
					Query(paramContainerID, containerId).
					Go(),
			))
		}
		containers = append(containers, VListGroup(
			web.Slot(
				VListItem(
					VListItemTitle(h.Text(groupName)),
				).Attr("v-bind", "props").Class("bg-light-blue-lighten-5"),
			).Name("activator").Scope(" {  props }"),
			h.Components(listItems...),
		).Value(groupName))
	}

	var cons []*Container
	var sharedGroups []h.HTMLComponent
	var sharedGroupNames []string

	b.db.Select("display_name,model_name,model_id").Where("shared = true AND locale_code = ?", locale).Group("display_name,model_name,model_id").Find(&cons)
	sort.Slice(cons, func(i, j int) bool {
		return b.ContainerByName(cons[i].ModelName).group != "" && b.ContainerByName(cons[j].ModelName).group == ""
	})
	for _, group := range utils.GroupBySlice[*Container, string](cons, func(builder *Container) string {
		return b.ContainerByName(builder.ModelName).group
	}) {
		if len(group) == 0 {
			break
		}
		var groupName = b.ContainerByName(group[0].ModelName).group
		if groupName == "" {
			groupName = "Others"
		}
		if b.expendContainers {
			sharedGroupNames = append(sharedGroupNames, groupName)
		}
		var listItems []h.HTMLComponent
		for _, builder := range group {
			c := b.ContainerByName(builder.ModelName)
			cover := c.cover
			if cover == "" {
				cover = path.Join(b.prefix, b.imagesPrefix, strings.ReplaceAll(c.name, " ", "")+".png")
			}
			containerName := i18n.T(ctx.R, presets.ModelsI18nModuleKey, c.name)
			listItems = append(listItems, VListItem(
				h.Div(
					VListItemTitle(h.Text(containerName)),
					VListItemSubtitle(VImg().Src(cover).Height(100)),
				).Attr("@click", web.Plaid().
					URL(fmt.Sprintf("%s/editors/%d?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
					EventFunc(AddContainerEvent).
					Query(presets.ParamID, ctx.R.FormValue(presets.ParamID)).
					Query(paramStatus, ctx.R.FormValue(paramStatus)).
					Query(paramPageVersion, pageVersion).
					Query(paramLocale, locale).
					Query(paramContainerName, builder.ModelName).
					Query(paramModelName, builder.ModelName).
					Query(paramModelID, builder.ModelID).
					Query(paramSharedContainer, "true").
					Query(paramContainerID, containerId).
					Go()),
			).Value(containerName))
		}

		sharedGroups = append(sharedGroups, VListGroup(
			web.Slot(
				VListItem(
					VListItemTitle(h.Text(groupName)),
				).Attr("v-bind", "props").Class("bg-light-blue-lighten-5"),
			).Name("activator").Scope(" {  props }"),
			h.Components(listItems...),
		).Value(groupName))

	}

	var backPlaid = web.Plaid().
		URL(fmt.Sprintf("%s/editors/%s?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
		EventFunc(ShowSortedContainerDrawerEvent).
		Query(presets.ParamID, pageID).
		Query(paramPageVersion, pageVersion).
		Query(paramLocale, locale).
		Query(paramStatus, ctx.R.FormValue(paramStatus)).
		Go()

	component = h.Components(h.Div(
		VBtn("").Size(SizeSmall).Icon("mdi-arrow-left").Variant(VariantText).
			Attr("@click", backPlaid),
		h.Span("Elements").Class("text-subtitle-1"),
	).Class("d-inline-flex pa-6 align-center"),
		VDivider(), web.Scope(
			VTabs(
				VTab(h.Text(msgr.New)).Value(msgr.New),
				VTab(h.Text(msgr.Shared)).Value(msgr.Shared),
			).Attr("v-model", "locals.tab").Class("px-6"),
			VTabsWindow(
				VTabsWindowItem(
					VList(containers...).Opened(groupsNames),
				).Value(msgr.New).Attr("style", "overflow-y: scroll; overflow-x: hidden; height: 610px;"),
				VTabsWindowItem(
					VList(sharedGroups...).Opened(sharedGroupNames),
				).Value(msgr.Shared).Attr("style", "overflow-y: scroll; overflow-x: hidden; height: 610px;"),
			).Attr("v-model", "locals.tab").Class("pa-6"),
		).Init(fmt.Sprintf(`{ tab : %s } `, msgr.New)).VSlot("{locals}"))
	return
}

func (b *Builder) AddContainerDialog(ctx *web.EventContext) (r web.EventResponse, err error) {
	pageID := ctx.QueryAsInt(paramPageID)
	pageVersion := ctx.R.FormValue(paramPageVersion)
	locale := ctx.R.FormValue(paramLocale)
	// okAction := web.Plaid().EventFunc(RenameContainerEvent).Query(paramContainerID, containerID).Go()
	msgr := i18n.MustGetModuleMessages(ctx.R, I18nPageBuilderKey, Messages_en_US).(*Messages)

	var containers []h.HTMLComponent
	for _, builder := range b.containerBuilders {
		cover := builder.cover
		if cover == "" {
			cover = path.Join(b.prefix, b.imagesPrefix, strings.ReplaceAll(builder.name, " ", "")+".png")
		}
		containers = append(containers,
			VCol(
				VCard(
					VImg().Src(cover).Height(200),
					VCardActions(
						VCardTitle(h.Text(i18n.T(ctx.R, presets.ModelsI18nModuleKey, builder.name))),
						VSpacer(),
						VBtn(msgr.Select).
							Variant(VariantText).
							Color("primary").Attr("@click",
							"dialogLocals.addContainerDialog = false;"+web.Plaid().
								URL(fmt.Sprintf("%s/editors/%d?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
								EventFunc(AddContainerEvent).
								Query(presets.ParamID, pageID).
								Query(paramPageVersion, pageVersion).
								Query(paramLocale, locale).
								Query(paramContainerName, builder.name).
								Go(),
						),
					),
				),
			).Cols(4),
		)
	}

	var cons []*Container
	err = b.db.Select("display_name,model_name,model_id").Where("shared = true AND locale_code = ?", locale).Group("display_name,model_name,model_id").Find(&cons).Error
	if err != nil {
		return
	}

	var sharedContainers []h.HTMLComponent
	for _, sharedC := range cons {
		c := b.ContainerByName(sharedC.ModelName)
		cover := c.cover
		if cover == "" {
			cover = path.Join(b.prefix, b.imagesPrefix, strings.ReplaceAll(c.name, " ", "")+".png")
		}
		sharedContainers = append(sharedContainers,
			VCol(
				VCard(
					VImg().Src(cover).Height(200),
					VCardActions(
						VCardTitle(h.Text(i18n.T(ctx.R, presets.ModelsI18nModuleKey, sharedC.DisplayName))),
						VSpacer(),
						VBtn(msgr.Select).
							Variant(VariantText).
							Color("primary").Attr("@click",
							"dialogLocals.addContainerDialog = false;"+web.Plaid().
								URL(fmt.Sprintf("%s/editors/%d?version=%s&locale=%s", b.prefix, pageID, pageVersion, locale)).
								EventFunc(AddContainerEvent).
								Query(presets.ParamID, pageID).
								Query(paramPageVersion, pageVersion).
								Query(paramLocale, locale).
								Query(paramContainerName, sharedC.ModelName).
								Query(paramModelID, sharedC.ModelID).
								Query(paramSharedContainer, "true").
								Go(),
						),
					),
				),
			).Cols(4),
		)
	}

	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
		Name: dialogPortalName,
		Body: web.Scope(
			VDialog(
				VTabs(
					VTab(h.Text(msgr.New)).Value(msgr.New),
					VTab(h.Text(msgr.Shared)).Value(msgr.Shared),
				).Attr("v-model", "dialogLocals.tab"),
				VWindow(
					VWindowItem(
						VSheet(
							VContainer(
								VRow(
									containers...,
								),
							),
						),
					).Value(msgr.New).Attr("style", "overflow-y: scroll; overflow-x: hidden; height: 610px;"),
					VWindowItem(
						VSheet(
							VContainer(
								VRow(
									sharedContainers...,
								),
							),
						),
					).Value(msgr.Shared).Attr("style", "overflow-y: scroll; overflow-x: hidden; height: 610px;"),
				).Attr("v-model", "dialogLocals.tab"),
			).Width("1200px").Attr("v-model", "dialogLocals.addContainerDialog"),
		).Init(fmt.Sprintf(`{addContainerDialog:true , tab : %s } `, msgr.New)).VSlot("{locals:dialogLocals}"),
	})

	return
}

type editorContainer struct {
	builder   *ContainerBuilder
	container *Container
}

func (b *Builder) getContainerBuilders(cs []*Container) (r []*editorContainer) {
	for _, c := range cs {
		for _, cb := range b.containerBuilders {
			if cb.name == c.ModelName {
				r = append(r, &editorContainer{
					builder:   cb,
					container: c,
				})
			}
		}
	}
	return
}

const (
	dialogPortalName = "pagebuilder_DialogPortalName"
)

func (b *Builder) pageEditorLayout(in web.PageFunc, config *presets.LayoutConfig) (out web.PageFunc) {
	return func(ctx *web.EventContext) (pr web.PageResponse, err error) {

		b.ps.InjectAssets(ctx)
		var innerPr web.PageResponse
		innerPr, err = in(ctx)
		if err != nil {
			panic(err)
		}
		pr.PageTitle = fmt.Sprintf("%s - %s", innerPr.PageTitle, "Page Builder")
		pr.Body = VApp(

			web.Portal().Name(presets.RightDrawerPortalName),
			web.Portal().Name(presets.DialogPortalName),
			web.Portal().Name(presets.DeleteConfirmPortalName),
			web.Portal().Name(dialogPortalName),
			innerPr.Body.(h.HTMLComponent),
		).Attr("id", "vt-app").
			Attr(web.VAssign("vars", `{presetsRightDrawer: false, presetsDialog: false, dialogPortalName: false}`)...)
		return
	}
}

func (b *Builder) ShowAddContainerDrawer(ctx *web.EventContext) (r web.EventResponse, err error) {
	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{Name: pageBuilderRightContentPortal, Body: b.renderContainersList(ctx)})
	return
}

func (b *Builder) ShowSortedContainerDrawer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var body h.HTMLComponent
	if body, err = b.renderContainersSortedList(ctx); err != nil {
		return
	}
	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{Name: pageBuilderRightContentPortal, Body: body})
	return
}

func (b *Builder) ShowEditContainerDrawer(ctx *web.EventContext) (r web.EventResponse, err error) {
	var body h.HTMLComponent
	if body, err = b.renderEditContainer(ctx); err != nil {
		return
	}
	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{Name: pageBuilderRightContentPortal, Body: body})
	return
}

func (b *Builder) ReloadRenderPageOrTemplate(ctx *web.EventContext) (r web.EventResponse, err error) {
	var body h.HTMLComponent
	body, _, err = b.renderPageOrTemplate(ctx, true)
	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{Name: editorPreviewContentPortal, Body: body.(*h.HTMLTagBuilder).Attr(web.VAssign("locals", "{el:$}")...)})
	return
}
