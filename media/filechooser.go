package media

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/qor5/admin/v3/media/base"
	"github.com/qor5/admin/v3/presets"

	"github.com/qor5/admin/v3/media/media_library"
	"github.com/qor5/web/v3"
	"github.com/qor5/x/v3/i18n"
	. "github.com/qor5/x/v3/ui/vuetify"
	h "github.com/theplant/htmlgo"
)

func fileChooser(mb *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nMediaLibraryKey, Messages_en_US).(*Messages)
		field := ctx.R.FormValue("field")
		cfg := stringToCfg(ctx.R.FormValue("cfg"))

		portalName := mainPortalName(field)
		r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
			Name: portalName,
			Body: VDialog(
				VCard(
					VToolbar(
						VBtn("").
							Icon("mdi-close").
							Theme(ThemeDark).
							Attr("@click", "vars.showFileChooser = false"),
						VToolbarTitle(msgr.ChooseAFile),
						VSpacer(),
						VLayout(
							VTextField().
								Variant(FieldVariantSoloInverted).
								PrependIcon("mdi-magnify").
								Label(msgr.Search).
								Flat(true).
								Clearable(true).
								HideDetails(true).
								ModelValue("").
								Attr("@keyup.enter", web.Plaid().
									EventFunc(imageSearchEvent).
									Query("field", field).
									FieldValue("cfg", h.JSONString(cfg)).
									FieldValue(searchKeywordName(field), web.Var("$event")).
									Go()),
						).Attr("style", "max-width: 650px"),
					).Color("primary").
						// MaxHeight(64).
						Flat(true).
						Theme(ThemeDark),
					web.Portal().Name(deleteConfirmPortalName(field)),
					web.Portal(
						fileChooserDialogContent(mb, field, ctx, cfg),
					).Name(dialogContentPortalName(field)),
				),
			).
				Fullscreen(true).
				// HideOverlay(true).
				Transition("dialog-bottom-transition").
				// Scrollable(true).
				Attr("v-model", "vars.showFileChooser"),
		})
		r.RunScript = `setTimeout(function(){ vars.showFileChooser = true }, 100)`
		return
	}
}

const (
	paramOrderByKey = "order_by"
	paramTypeKey    = "type"
	paramTab        = "tab"
	paramParentID   = "parent_id"

	orderByCreatedAt     = "created_at"
	orderByCreatedAtDESC = "created_at_desc"

	typeAll   = "all"
	typeImage = "image"
	typeVideo = "video"
	typeFile  = "file"

	tabFiles   = "files"
	tabFolders = "folders"
)

type selectItem struct {
	Text  string
	Value string
}

func fileChooserDialogContent(mb *Builder, field string, ctx *web.EventContext,
	cfg *media_library.MediaBoxConfig,
) h.HTMLComponent {
	var (
		db         = mb.db
		msgr       = i18n.MustGetModuleMessages(ctx.R, I18nMediaLibraryKey, Messages_en_US).(*Messages)
		keyword    = ctx.Param(searchKeywordName(field))
		tab        = ctx.Param(paramTab)
		orderByVal = ctx.Param(paramOrderByKey)
		typeVal    = ctx.Param(paramTypeKey)
		parentID   = ctx.ParamAsInt(paramParentID)

		wh    = db.Model(&media_library.MediaLibrary{})
		files []*media_library.MediaLibrary
		bc    h.HTMLComponent
	)
	if tab == "" {
		tab = tabFiles
	}

	if mb.searcher != nil {
		wh = mb.searcher(wh, ctx)
	} else if mb.currentUserID != nil {
		wh = wh.Where("user_id = ? ", mb.currentUserID(ctx))
	}
	switch orderByVal {
	case orderByCreatedAt:
		wh = wh.Order("created_at")
	default:
		orderByVal = orderByCreatedAtDESC
		wh = wh.Order("created_at DESC")
	}

	switch typeVal {
	case typeImage:
		wh = wh.Where("selected_type = ?", media_library.ALLOW_TYPE_IMAGE)
	case typeVideo:
		wh = wh.Where("selected_type = ?", media_library.ALLOW_TYPE_VIDEO)
	case typeFile:
		wh = wh.Where("selected_type = ?", media_library.ALLOW_TYPE_FILE)
	default:
		typeVal = typeAll
	}
	if tab == tabFiles {
		wh = wh.Where("folder = ?", false)
	} else {
		wh = wh.Where("parent_id = ?", parentID)
		items := parentFolders(ctx, mb.db, uint(parentID), uint(parentID), nil)
		bc = VBreadcrumbs(
			items...,
		)
	}

	currentPageInt, _ := strconv.Atoi(ctx.R.FormValue(currentPageName(field)))
	if currentPageInt == 0 {
		currentPageInt = 1
	}

	if len(cfg.Sizes) > 0 {
		cfg.AllowType = media_library.ALLOW_TYPE_IMAGE
	}

	if len(cfg.AllowType) > 0 {
		wh = wh.Where("selected_type = ?", cfg.AllowType)
	}

	if len(keyword) > 0 {
		wh = wh.Where("file ILIKE ?", fmt.Sprintf("%%%s%%", keyword))
	}

	var count int64
	err := wh.Count(&count).Error
	if err != nil {
		panic(err)
	}
	perPage := mb.mediaLibraryPerPage
	pagesCount := int(count/int64(perPage) + 1)
	if count%int64(perPage) == 0 {
		pagesCount--
	}

	wh = wh.Limit(perPage).Offset((currentPageInt - 1) * perPage)
	err = wh.Find(&files).Error
	if err != nil {
		panic(err)
	}

	fileAccept := "*/*"
	if cfg.AllowType == media_library.ALLOW_TYPE_IMAGE {
		fileAccept = "image/*"
	}

	row := VRow(
		h.If(mb.uploadIsAllowed(ctx.R) == nil,
			VCol(
				VCard(
					VProgressCircular().
						Color("primary").
						Indeterminate(true),
				).
					Class("d-flex align-center justify-center").
					Height(200),
			).
				Attr("v-for", "f in locals.fileChooserUploadingFiles").
				Cols(6).Sm(4).Md(3),
		),
	)

	initCroppingVars := []string{fileCroppingVarName(0) + ": false"}

	for _, f := range files {
		var fileComp h.HTMLComponent
		fileComp = fileOrFolderComponent(mb, field, ctx, f, msgr, cfg, initCroppingVars)
		row.AppendChildren(
			VCol(fileComp).Cols(6).Sm(4).Md(3),
		)
	}

	return h.Div(
		web.Portal().Name(newFolderDialogPortalName),
		web.Portal().Name(moveToFolderDialogPortalName),
		imageDialog(),
		VSnackbar(h.Text(msgr.DescriptionUpdated)).
			Attr("v-model", "vars.snackbarShow").
			Location("top").
			Color("primary").
			Timeout(5000),
		web.Scope(
			VContainer(
				h.If(field == mediaLibraryListField,
					VRow(
						VCol(
							web.Scope(
								VTabs(
									VTab(h.Text(msgr.Files)).Value(tabFiles),
									VTab(h.Text("Folders")).Value(tabFolders),
								).Attr("v-model", "tabLocals.tab").
									Attr("@update:model-value",
										fmt.Sprintf(`$event=="%s"?null:%v`, tab, web.Plaid().PushState(true).Query(paramTab, web.Var("$event")).Go()),
									),
							).VSlot(`{locals:tabLocals}`).Init(fmt.Sprintf(`{tab:"%s"}`, tab)),
						),
						VSpacer(),
						VCol(
							VSelect().Items([]selectItem{
								{Text: msgr.All, Value: typeAll},
								{Text: msgr.Images, Value: typeImage},
								{Text: msgr.Videos, Value: typeVideo},
								{Text: msgr.Files, Value: typeFile},
							}).ItemTitle("Text").ItemValue("Value").
								Attr(web.VField(paramTypeKey, typeVal)...).
								Attr("@change",
									web.GET().PushState(true).
										Query(paramTypeKey, web.Var("$event")).
										MergeQuery(true).Go(),
								).
								Density(DensityCompact).Variant(FieldVariantSolo),
						).Cols(3),
						VCol(
							VSelect().Items([]selectItem{
								{Text: msgr.UploadedAtDESC, Value: orderByCreatedAtDESC},
								{Text: msgr.UploadedAt, Value: orderByCreatedAt},
							}).ItemTitle("Text").ItemValue("Value").
								Attr(web.VField(paramOrderByKey, orderByVal)...).
								Attr("@change",
									web.GET().PushState(true).
										Query(paramOrderByKey, web.Var("$event")).
										MergeQuery(true).Go(),
								).
								Density(DensityCompact).Variant(FieldVariantSolo),
						).Cols(3),
						VCol(
							h.If(
								tab == tabFolders,
								VBtn("New Folder").PrependIcon("mdi-plus").
									Variant(VariantOutlined).Class("mr-2").
									Attr("@click",
										web.Plaid().EventFunc(NewFolderDialogEvent).
											Query(paramParentID, ctx.Param(paramParentID)).Go()),
							),
							h.If(mb.uploadIsAllowed(ctx.R) == nil,
								h.Div(
									VBtn("Upload file").PrependIcon("mdi-upload").Color(ColorSecondary).
										Attr("@click", "$refs.uploadInput.click()"),
									h.Input("").
										Attr("ref", "uploadInput").
										Attr("accept", fileAccept).
										Type("file").
										Attr("multiple", true).
										Style("display:none").
										Attr("@change",
											"form.NewFiles = [...$event.target.files];"+
												web.Plaid().
													BeforeScript("locals.fileChooserUploadingFiles = $event.target.files").
													EventFunc(uploadFileEvent).
													Query("field", field).
													FieldValue("cfg", h.JSONString(cfg)).
													Go()),
								),
							),
						).Class("d-inline-flex"),
					).Justify("end"),
				),
				VRow(
					VCol(bc),
				),
				row,
				VRow(
					VCol().Cols(1),
					VCol(
						VPagination().
							Length(pagesCount).
							ModelValue(int(currentPageInt)).
							Attr("@input", web.Plaid().
								FieldValue(currentPageName(field), web.Var("$event")).
								EventFunc(imageJumpPageEvent).
								Query("field", field).
								FieldValue("cfg", h.JSONString(cfg)).
								Go()),
					).Cols(10),
				),
				VCol().Cols(1),
				VRow(
					VCol(
						h.Text(fmt.Sprintf("{{locals.select_ids.length}} %s", "Selected")),
						VBtn("Move to").Size(SizeSmall).Variant(VariantOutlined).
							Attr(":disabled", "locals.select_ids.length==0").
							Color(ColorSecondary).Class("ml-4").
							Attr("@click", web.Plaid().EventFunc(MoveToFolderDialogEvent).Query(ParamSelectIDS, web.Var(`locals.select_ids.join(",")`)).Go()),
						VBtn("Delete").Size(SizeSmall).Variant(VariantOutlined).Color(ColorWarning).Class("ml-2"),
					),
				).Attr("v-if", "locals.select_ids && locals.select_ids.length>0"),
			).Fluid(true),
		).Init(fmt.Sprintf(`{fileChooserUploadingFiles: [], %s}`, strings.Join(initCroppingVars, ", "))).
			VSlot("{ locals}").Init("{select_ids:[]}"),
		VOverlay(
			h.Img("").Attr(":src", "vars.isImage? vars.mediaShow: ''").
				Style("max-height: 80vh; max-width: 80vw; background: rgba(0, 0, 0, 0.5)"),
			h.Div(
				h.A(
					VIcon("info").Size(SizeSmall).Class("mb-1"),
					h.Text("{{vars.mediaName}}"),
				).Attr(":href", "vars.mediaShow? vars.mediaShow: ''").Target("_blank").
					Class("white--text").Style("text-decoration: none;"),
			).Class("d-flex align-center justify-center pt-2"),
		).Attr("v-if", "vars.mediaName").Attr("@click", "vars.mediaName = null").ZIndex(10),
	).Attr(web.VAssign("vars", `{snackbarShow: false, mediaShow: null, mediaName: null, isImage: false,moving:false,imagePreview:false,imageSrc:""}`)...)
}

func fileChips(f *media_library.MediaLibrary) h.HTMLComponent {
	text := "original"
	if f.File.Width != 0 && f.File.Height != 0 {
		text = fmt.Sprintf("%s(%dx%d)", "original", f.File.Width, f.File.Height)
	}
	if f.File.FileSizes["original"] != 0 {
		text = fmt.Sprintf("%s %s", text, base.ByteCountSI(f.File.FileSizes["original"]))
	}
	return h.Text(text)
}

type uploadFiles struct {
	NewFiles []*multipart.FileHeader
}

func uploadFile(mb *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		field := ctx.R.FormValue("field")
		cfg := stringToCfg(ctx.R.FormValue("cfg"))

		if err = mb.uploadIsAllowed(ctx.R); err != nil {
			return
		}

		var uf uploadFiles
		ctx.MustUnmarshalForm(&uf)
		for _, fh := range uf.NewFiles {
			m := media_library.MediaLibrary{}

			if base.IsImageFormat(fh.Filename) {
				m.SelectedType = media_library.ALLOW_TYPE_IMAGE
			} else if base.IsVideoFormat(fh.Filename) {
				m.SelectedType = media_library.ALLOW_TYPE_VIDEO
			} else {
				m.SelectedType = media_library.ALLOW_TYPE_FILE
			}
			err = m.File.Scan(fh)
			if err != nil {
				panic(err)
			}
			if mb.currentUserID != nil {
				m.UserID = mb.currentUserID(ctx)
			}
			err = base.SaveUploadAndCropImage(mb.db, &m)
			if err != nil {
				presets.ShowMessage(&r, err.Error(), "error")
				return r, nil
			}
		}

		renderFileChooserDialogContent(ctx, &r, field, mb, cfg)
		return
	}
}

func mergeNewSizes(m *media_library.MediaLibrary, cfg *media_library.MediaBoxConfig) (sizes map[string]*base.Size, r bool) {
	sizes = make(map[string]*base.Size)
	for k, size := range cfg.Sizes {
		if m.File.Sizes[k] != nil {
			sizes[k] = m.File.Sizes[k]
			continue
		}
		sizes[k] = size
		r = true
	}
	return
}

func chooseFile(mb *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		db := mb.db
		field := ctx.R.FormValue("field")
		id := ctx.ParamAsInt(mediaID)
		cfg := stringToCfg(ctx.R.FormValue("cfg"))

		var m media_library.MediaLibrary
		err = db.Find(&m, id).Error
		if err != nil {
			return
		}
		sizes, needCrop := mergeNewSizes(&m, cfg)

		if needCrop {
			err = m.ScanMediaOptions(media_library.MediaOption{
				Sizes: sizes,
				Crop:  true,
			})
			if err != nil {
				return
			}
			err = db.Save(&m).Error
			if err != nil {
				return
			}

			err = base.SaveUploadAndCropImage(db, &m)
			if err != nil {
				presets.ShowMessage(&r, err.Error(), "error")
				return r, nil
			}
		}

		mediaBox := media_library.MediaBox{
			ID:          json.Number(fmt.Sprint(m.ID)),
			Url:         m.File.Url,
			VideoLink:   "",
			FileName:    m.File.FileName,
			Description: m.File.Description,
			FileSizes:   m.File.FileSizes,
			Width:       m.File.Width,
			Height:      m.File.Height,
		}

		r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
			Name: mediaBoxThumbnailsPortalName(field),
			Body: mediaBoxThumbnails(ctx, &mediaBox, field, cfg, false, false),
		})
		r.RunScript = `vars.showFileChooser = false`
		return
	}
}

func searchFile(mb *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		field := ctx.R.FormValue("field")
		cfg := stringToCfg(ctx.R.FormValue("cfg"))

		ctx.R.Form[currentPageName(field)] = []string{"1"}

		renderFileChooserDialogContent(ctx, &r, field, mb, cfg)
		return
	}
}

func jumpPage(mb *Builder) web.EventFunc {
	return func(ctx *web.EventContext) (r web.EventResponse, err error) {
		field := ctx.R.FormValue("field")
		cfg := stringToCfg(ctx.R.FormValue("cfg"))
		renderFileChooserDialogContent(ctx, &r, field, mb, cfg)
		return
	}
}

func renderFileChooserDialogContent(ctx *web.EventContext, r *web.EventResponse, field string, mb *Builder, cfg *media_library.MediaBoxConfig) {
	r.UpdatePortals = append(r.UpdatePortals, &web.PortalUpdate{
		Name: dialogContentPortalName(field),
		Body: fileChooserDialogContent(mb, field, ctx, cfg),
	})
}

func fileComponent(
	mb *Builder,
	field string,
	ctx *web.EventContext,
	f *media_library.MediaLibrary,
	msgr *Messages,
	cfg *media_library.MediaBoxConfig,
	initCroppingVars []string,
	event *string,
) (title h.HTMLComponent, content h.HTMLComponent) {
	_, needCrop := mergeNewSizes(f, cfg)
	croppingVar := fileCroppingVarName(f.ID)
	initCroppingVars = append(initCroppingVars, fmt.Sprintf("%s: false", croppingVar))
	imgClickVars := fmt.Sprintf("vars.mediaShow = '%s'; vars.mediaName = '%s'; vars.isImage = %s", f.File.URL(), f.File.FileName, strconv.FormatBool(base.IsImageFormat(f.File.FileName)))

	src := f.File.URL("original")
	*event = fmt.Sprintf(`console.log(2);vars.imageSrc="%s";vars.imagePreview=true;`, src)
	title = h.Div(
		h.If(
			base.IsImageFormat(f.File.FileName),
			VImg(
				h.If(needCrop,
					h.Div(
						VProgressCircular().Indeterminate(true),
						h.Span(msgr.Cropping).Class("text-h6 pl-2"),
					).Class("d-flex align-center justify-center v-card--reveal white--text").
						Style("height: 100%; background: rgba(0, 0, 0, 0.5)").
						Attr("v-if", fmt.Sprintf("locals.%s", croppingVar)),
				),
			).Src(src).Height(120).Cover(true),
		).Else(
			fileThumb(f.File.FileName),
		),
	).AttrIf("role", "button", field != mediaLibraryListField).
		AttrIf("@click", web.Plaid().
			BeforeScript(fmt.Sprintf("locals.%s = true", croppingVar)).
			EventFunc(chooseFileEvent).
			Query("field", field).
			Query(mediaID, fmt.Sprint(f.ID)).
			FieldValue("cfg", h.JSONString(cfg)).
			Go(), field != mediaLibraryListField).
		AttrIf("@click", imgClickVars, field == mediaLibraryListField)

	content = h.Components(
		web.Slot(
			web.Scope(
				VTextField().Attr(web.VField("name", f.File.FileName)...).
					Attr(":variant", fmt.Sprintf(`locals.edit_%v?"%s":"%s"`, f.ID, VariantOutlined, VariantPlain)),
			).VSlot(`{form}`),
		).Name("title"),
		web.Slot(h.If(base.IsImageFormat(f.File.FileName),
			fileChips(f))).Name("subtitle"),
	)

	return
}

func fileOrFolderComponent(
	mb *Builder,
	field string,
	ctx *web.EventContext,
	f *media_library.MediaLibrary,
	msgr *Messages,
	cfg *media_library.MediaBoxConfig,
	initCroppingVars []string,
) h.HTMLComponent {
	var (
		title, content            h.HTMLComponent
		checkEvent                = fmt.Sprintf(`console.log(1);let arr=locals.select_ids;let find_id=%v; arr.includes(find_id)?arr.splice(arr.indexOf(find_id), 1):arr.push(find_id);`, f.ID)
		clickCardWithoutMoveEvent = "null"
	)
	menus := h.Components(
		VListItem(h.Text("Rename")).Attr("@click", fmt.Sprintf("locals.edit_%v=true", f.ID)),
		VListItem(h.Text("Move to")).Attr("@click", fmt.Sprintf("locals.select_ids.push(%v)", f.ID)),
		h.If(mb.deleteIsAllowed(ctx.R, f) == nil, VListItem(h.Text(msgr.Delete)).Attr("@click",
			web.Plaid().
				EventFunc(deleteConfirmationEvent).
				Query("field", field).
				Query(mediaID, fmt.Sprint(f.ID)).
				Go())),
	)

	if f.Folder {
		title, content = folderComponent(mb, field, ctx, f, msgr)
		clickCardWithoutMoveEvent = web.Plaid().PushState(true).MergeQuery(true).Query(paramParentID, f.ID).Go()
	} else {
		title, content = fileComponent(mb, field, ctx, f, msgr, cfg, initCroppingVars, &clickCardWithoutMoveEvent)
	}

	return VCard(
		VCheckbox().
			Attr(":model-value", fmt.Sprintf(`locals.select_ids.includes(%v)`, f.ID)).
			Attr("@update:model-value", checkEvent).
			Attr("style", "z-index:2").
			Class("position-absolute top-0 right-0").Attr("v-if", "locals.select_ids.length>0"),
		VCardText(
			VCard(
				title,
			).Height(120).Elevation(0),
		).Class("pa-0", W100),
		VCardItem(
			VCard(
				content,
				web.Slot(
					VMenu(
						web.Slot(
							VBtn("").Children(
								VIcon("mdi-dots-horizontal"),
							).Attr("v-bind", "props").Variant(VariantText).Size(SizeSmall),
						).Name("activator").Scope("{ props }"),
						VList(
							menus...,
						),
					),
				).Name(VSlotAppend),
			).Color(ColorGreyLighten5),
		).Class("pa-0"),
	).Class("position-relative").
		Hover(true).
		Attr("@click", fmt.Sprintf("locals.select_ids.length>0?function(){%s}():%s", checkEvent, clickCardWithoutMoveEvent))
}

func folderComponent(
	mb *Builder,
	field string,
	ctx *web.EventContext,
	f *media_library.MediaLibrary,
	msgr *Messages,
) (title h.HTMLComponent, content h.HTMLComponent) {
	var count int64
	mb.db.Model(media_library.MediaLibrary{}).Where("parent_id = ?", f.ID).Count(&count)
	title = VCardText(VIcon("mdi-folder").Size(90).Color(ColorPrimary)).Class("d-flex justify-center align-center")
	content = h.Components(
		web.Slot(
			web.Scope(
				VTextField().Attr(web.VField("name", f.File.FileName)...).
					Attr(":variant", fmt.Sprintf(`locals.edit_%v?"%s":"%s"`, f.ID, VariantOutlined, VariantPlain)),
			).VSlot(`{form}`),
		).Name("title"),
		web.Slot(h.Text(fmt.Sprintf("%v items", count))).Name("subtitle"),
	)

	return
}

func parentFolders(ctx *web.EventContext, db *gorm.DB, currentID, parentID uint, existed map[uint]bool) (comps h.HTMLComponents) {
	if existed == nil {
		existed = make(map[uint]bool)
	}
	var (
		item    *VBreadcrumbsItemBuilder
		current *media_library.MediaLibrary
	)
	if err := db.First(&current, currentID).Error; err != nil {
		return
	}
	item = VBreadcrumbsItem().Title(current.File.FileName)
	if currentID == parentID {
		item.Disabled(true)
	} else {
		item.Href("#").Attr("@click.prevent", web.Plaid().PushState(true).MergeQuery(true).Query(paramParentID, currentID).Go())
	}
	comps = append(comps, item)
	if current.ParentId == 0 || existed[current.ID] {
		comps = append(h.Components(VBreadcrumbsItem().Title("/").Href("#").Attr("@click.prevent", web.Plaid().PushState(true).MergeQuery(true).Query(paramParentID, 0).Go())), comps...)

		return
	}
	comps = append(h.Components(h.Text("/")), comps...)
	existed[currentID] = true
	return append(parentFolders(ctx, db, current.ParentId, parentID, existed), comps...)
}

func imageDialog() h.HTMLComponent {
	return VDialog(
		VCard(
			VBtn("").Icon("mdi-close").
				Variant(VariantText).Attr("@click", "vars.imagePreview=false").
				Class("position-absolute right-0 top-0").Attr("style", "z-index:2"),
			VImg().Attr(":src", "vars.imageSrc").Width(658),
		).Class("position-relative"),
	).MaxWidth(658).Attr("v-model", "vars.imagePreview")
}
