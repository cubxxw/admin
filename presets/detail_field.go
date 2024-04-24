package presets

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/qor5/admin/v3/presets/actions"
	. "github.com/qor5/ui/v3/vuetify"
	"github.com/qor5/web/v3"
	"github.com/sunfmin/reflectutils"
	h "github.com/theplant/htmlgo"
)

const (
	DetailFieldName = "detailField"

	detailListFieldEditBtnKey   = "detailListFieldEditBtn"
	detailListFieldSaveBtnKey   = "detailListFieldSaveBtn"
	detailListFieldDeleteBtnKey = "detailListFieldDeleteBtn"

	detailListFieldEditing = "detailListFieldEditing"
)

type DetailFieldsBuilder struct {
	mb           *ModelBuilder
	detailFields []*DetailFieldBuilder
	FieldsBuilder
}

func (d *DetailFieldsBuilder) GetDetailField(name string) *DetailFieldBuilder {
	for _, v := range d.detailFields {
		if v.name == name {
			return v
		}
	}
	return nil
}

func (d *DetailFieldsBuilder) appendNewDetailFieldWithName(name string) (r *DetailFieldBuilder) {
	r = &DetailFieldBuilder{
		NameLabel: NameLabel{
			name:  name,
			label: name,
		},
		switchable:        false,
		saver:             nil,
		setter:            nil,
		componentShowFunc: nil,
		componentEditFunc: nil,
		father:            d,
		elementShowFunc:   nil,
		elementEditFunc:   nil,
		design: &DetailFieldDesign{
			disableElementDeleteBtn: false,
			disableElementCreateBtn: false,
		},
		isList: false,
	}
	r.FieldsBuilder.Model(d.mb.model)
	r.saver = r.DefaultSaveFunc

	d.detailFields = append(d.detailFields, r)

	d.Field(name).ComponentFunc(func(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
		panic("you must set ShowComponentFunc and EditComponentFunc if you want to use DetailFieldsBuilder")
	})

	return
}

type DetailFieldDesign struct {
	disableElementDeleteBtn bool
	disableElementCreateBtn bool
}

func (d *DetailFieldDesign) DisableElementDeleteBtn() *DetailFieldDesign {
	d.disableElementDeleteBtn = true
	return d
}

func (d *DetailFieldDesign) DisableElementCreateBtn() *DetailFieldDesign {
	d.disableElementCreateBtn = true
	return d
}

// DetailFieldBuilder
// save: 	   fetcher => setter => saver
// show, edit: fetcher => setter
type DetailFieldBuilder struct {
	NameLabel
	// if the field can switch status to edit and show, switchable must be true
	switchable        bool
	saver             SaveFunc
	setter            SetterFunc
	componentShowFunc FieldComponentFunc
	componentEditFunc FieldComponentFunc
	father            *DetailFieldsBuilder
	design            *DetailFieldDesign
	FieldsBuilder

	// only used if isList = true
	isList             bool
	elementShowFunc    FieldComponentFunc
	elementEditFunc    FieldComponentFunc
	elementUnmarshaler func(toObj, formObj any, prefix string, ctx *web.EventContext) error
}

func (b *DetailFieldBuilder) IsList(v interface{}) (r *DetailFieldBuilder) {
	if b.father.model == nil {
		panic("model must be provided")
	}
	rt := reflectutils.GetType(b.father.model, b.name)
	if rt.Kind() != reflect.Slice {
		panic("field kind must be slice")
	}
	if reflect.TypeOf(reflect.New(rt.Elem()).Elem().Interface()) != reflect.TypeOf(v) {
		panic(fmt.Sprintf("%s not equal to %s", reflect.New(rt.Elem()).Elem().Interface(), reflect.TypeOf(v)))
	}

	r = b
	r.FieldsBuilder.Model(v)
	r.isList = true
	r.saver = r.DefaultListElementSaveFunc

	return
}

func (b *DetailFieldBuilder) SetSwitchable(v bool) (r *DetailFieldBuilder) {
	r = b
	b.switchable = v
	return
}

// Editing default saver only save these field
func (b *DetailFieldBuilder) Editing(fields ...interface{}) (r *DetailFieldBuilder) {
	r = b
	b.FieldsBuilder = *b.FieldsBuilder.Only(fields...)
	return
}

func (b *DetailFieldBuilder) SaveFunc(v SaveFunc) (r *DetailFieldBuilder) {
	if v == nil {
		panic("value required")
	}
	b.saver = v
	return b
}

func (b *DetailFieldBuilder) SetterFunc(v SetterFunc) (r *DetailFieldBuilder) {
	if v == nil {
		panic("value required")
	}
	b.setter = v
	return b
}

func (b *DetailFieldBuilder) ElementUnmarshalFunc(v func(toObj, formObj any, prefix string, ctx *web.EventContext) error) (r *DetailFieldBuilder) {
	if v == nil {
		panic("value required")
	}
	b.elementUnmarshaler = v
	return b
}

func (b *DetailFieldBuilder) ShowComponentFunc(v FieldComponentFunc) (r *DetailFieldBuilder) {
	if !b.switchable {
		panic("detailField is not switchable")
	}
	if v == nil {
		panic("value required")
	}
	b.componentShowFunc = v
	if b.componentEditFunc != nil {
		b.ComponentFunc(func(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
			return b.showComponent(obj, field, ctx)
		})
	}
	return b
}

func (b *DetailFieldBuilder) EditComponentFunc(v FieldComponentFunc) (r *DetailFieldBuilder) {
	if !b.switchable {
		panic("detailField is not switchable")
	}
	if v == nil {
		panic("value required")
	}
	b.componentEditFunc = v
	if b.componentShowFunc != nil {
		b.ComponentFunc(func(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
			return b.showComponent(obj, field, ctx)
		})
	}
	return b
}

func (b *DetailFieldBuilder) Label(label string) (r *DetailFieldBuilder) {
	b.father.Field(b.name).Label(label)
	return b
}

func (b *DetailFieldBuilder) FieldDesign() (r *DetailFieldDesign) {
	return b.design
}

func (b *DetailFieldBuilder) ElementShowComponentFunc(v FieldComponentFunc) (r *DetailFieldBuilder) {
	if !b.switchable {
		panic("detailField is not switchable")
	}
	if v == nil {
		panic("value required")
	}
	b.elementShowFunc = v
	if b.elementEditFunc != nil {
		b.ComponentFunc(func(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
			return b.listComponent(obj, field, ctx, -1, -1, -1)
		})
	}
	return b
}

func (b *DetailFieldBuilder) ElementEditComponentFunc(v FieldComponentFunc) (r *DetailFieldBuilder) {
	if !b.switchable {
		panic("detailField is not switchable")
	}
	if v == nil {
		panic("value required")
	}
	b.elementEditFunc = v
	if b.elementShowFunc != nil {
		b.ComponentFunc(func(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
			return b.listComponent(obj, field, ctx, -1, -1, -1)
		})
	}
	return b
}

// ComponentFunc set FieldBuilder compFunc
func (b *DetailFieldBuilder) ComponentFunc(v FieldComponentFunc) (r *FieldBuilder) {
	r = b.father.Field(b.name)
	return r.ComponentFunc(v)
}

func (b *DetailFieldBuilder) ListFieldPrefix(index int) string {
	return fmt.Sprintf("%s[%b]", b.name, index)
}

func (b *DetailFieldBuilder) showComponent(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
	btn := VBtn("").Size(SizeXSmall).Variant("text").
		Rounded("0").
		Icon("mdi-square-edit-outline").
		Attr("v-if", fmt.Sprintf(`locals.showBtn`)).
		Attr("@click", web.Plaid().EventFunc(actions.DoEditDetailingField).
			Query(DetailFieldName, b.name).
			Query(ParamID, ctx.Queries().Get(ParamID)).
			Go())

	return web.Portal(
		web.Scope(
			web.Scope(
				h.Div(
					VCard(
						VCardText(
							h.Div(
								// detailFields
								h.Div(b.componentShowFunc(obj, field, ctx)).
									Class("flex-grow-1 pr-3"),
								// edit btn
								h.Div(btn).Style("width:32px;"),
							).Class("d-flex justify-space-between").Style("color:initial;"),
						),
					).Variant(VariantOutlined).Color("grey").
						Attr("@mouseenter", fmt.Sprintf(`locals.showBtn = true`)).
						Attr("@mouseleave", fmt.Sprintf(`locals.showBtn = false`)),
				),
			).VSlot("{ locals }").Init(`{ showBtn:false,color:blue }`),
		).VSlot("{ form }"),
	).Name(b.FieldPortalName())
}

func (b *DetailFieldBuilder) editComponent(obj interface{}, field *FieldContext, ctx *web.EventContext) h.HTMLComponent {
	btn := VBtn("Save").Size(SizeSmall).
		Attr("style", "text-transform: none;").
		Attr("@click", web.Plaid().EventFunc(actions.DoSaveDetailingField).
			Query(DetailFieldName, b.name).
			Query(ParamID, ctx.Queries().Get(ParamID)).
			Go())

	return web.Portal(
		web.Scope(
			web.Scope(
				h.Div(
					VCard(
						VCardText(
							h.Div(
								// detailFields
								h.Div(b.componentEditFunc(obj, field, ctx)).
									Class("flex-grow-1"),
								// save btn
								h.Div(btn).Class("align-self-end"),
							).Class("d-flex flex-column").Style("color:initial;"),
						),
					).Variant(VariantOutlined).Color("grey").
						Attr("@mouseenter", fmt.Sprintf(`locals.showBtn = true`)).
						Attr("@mouseleave", fmt.Sprintf(`locals.showBtn = false`)), h.Br(),
				),
			).VSlot("{ locals }").Init(`{ showBtn:false }`),
		).VSlot("{ form }"),
	).Name(b.FieldPortalName())
}

func (b *DetailFieldBuilder) DefaultSaveFunc(obj interface{}, id string, ctx *web.EventContext) (err error) {
	if tf := reflect.TypeOf(obj).Kind(); tf != reflect.Ptr {
		return errors.New(fmt.Sprintf("model %#+v must be pointer", obj))
	}
	var formObj = reflect.New(reflect.TypeOf(obj).Elem()).Interface()

	if err = b.DefaultElementUnmarshal(obj, formObj, b.name, ctx); err != nil {
		return
	}

	err = b.father.mb.p.dataOperator.Save(obj, id, ctx)
	return
}

func (b *DetailFieldBuilder) DefaultListElementSaveFunc(obj interface{}, id string, ctx *web.EventContext) (err error) {
	// Delete or Add row
	if ctx.Queries().Get(b.SaveBtnKey()) == "" {
		err = b.father.mb.p.dataOperator.Save(obj, id, ctx)
		return
	}

	var index int64
	index, err = strconv.ParseInt(ctx.Queries().Get(b.SaveBtnKey()), 10, 64)
	if err != nil {
		return
	}

	listObj := reflect.ValueOf(reflectutils.MustGet(obj, b.name))
	elementObj := listObj.Index(int(index)).Interface()
	formObj := reflect.New(reflect.TypeOf(b.model).Elem()).Interface()
	if err = b.DefaultElementUnmarshal(elementObj, formObj, b.ListFieldPrefix(int(index)), ctx); err != nil {
		return
	}
	listObj.Index(int(index)).Set(reflect.ValueOf(elementObj))

	err = b.father.mb.p.dataOperator.Save(obj, id, ctx)
	return
}

func (b *DetailFieldBuilder) listComponent(obj interface{}, field *FieldContext, ctx *web.EventContext, deletedID, editID, saveID int) h.HTMLComponent {
	list, err := reflectutils.Get(obj, b.name)
	if err != nil {
		panic(err)
	}

	rows := h.Div()

	if b.label != "" {
		rows.AppendChildren(h.Div(h.Span(b.label).Style("color:black; fontSize:16px; font-weight:500;")).Style("margin-bottom:8px;"))
	}

	if list != nil {
		i := 0
		reflectutils.ForEach(list, func(elementObj interface{}) {
			defer func() { i++ }()
			// set fieldSetting to ctx.R.Form by sortIndex
			sortIndex := i
			// find fieldSetting from ctx.R.Form by fromIndex
			fromIndex := i
			if deletedID != -1 && i >= deletedID {
				// if last event is click deleteBtn, fromIndex should add one
				fromIndex++
			}
			if editID == sortIndex {
				// if click edit
				rows.AppendChildren(b.editElement(elementObj, sortIndex, fromIndex, ctx))
			} else if saveID == sortIndex {
				// if click save
				rows.AppendChildren(b.showElement(elementObj, sortIndex, ctx))
			} else {
				// default
				isEditing := ctx.R.FormValue(b.ListElementIsEditing(fromIndex)) != ""
				if !isEditing {
					rows.AppendChildren(b.showElement(elementObj, sortIndex, ctx))
				} else {
					formObj := reflect.New(reflect.TypeOf(b.model).Elem()).Interface()
					if b.elementUnmarshaler != nil {
						err = b.elementUnmarshaler(elementObj, formObj, b.ListFieldPrefix(fromIndex), ctx)
					} else {
						err = b.DefaultElementUnmarshal(elementObj, formObj, b.ListFieldPrefix(fromIndex), ctx)
					}
					if err != nil {
						panic(err)
					}
					rows.AppendChildren(b.editElement(elementObj, sortIndex, fromIndex, ctx))
				}
			}
		})
	}

	if !b.design.disableElementCreateBtn {
		addBtn := VBtn("Add row").
			Attr("@click", web.Plaid().EventFunc(actions.DoCreateDetailingListField).
				Query(DetailFieldName, b.name).
				Query(ParamID, ctx.Queries().Get(ParamID)).
				Go())
		rows.AppendChildren(addBtn)
	}

	return web.Portal(
		web.Scope(rows).VSlot("{ form }"),
	).Name(b.FieldPortalName())
}

func (b *DetailFieldBuilder) EditBtnKey() string {
	return fmt.Sprintf("%s_%s", detailListFieldEditBtnKey, b.name)
}

func (b *DetailFieldBuilder) SaveBtnKey() string {
	return fmt.Sprintf("%s_%s", detailListFieldSaveBtnKey, b.name)
}

func (b *DetailFieldBuilder) DeleteBtnKey() string {
	return fmt.Sprintf("%s_%s", detailListFieldDeleteBtnKey, b.name)
}

func (b *DetailFieldBuilder) ListElementIsEditing(index int) string {
	return fmt.Sprintf("%s_%s[%b].%s", deletedHiddenNamePrefix, b.name, index, detailListFieldEditing)
}

func (b *DetailFieldBuilder) ListElementPortalName(index int) string {
	return fmt.Sprintf("DetailElementPortal_%s_%b", b.name, index)
}

func (b *DetailFieldBuilder) FieldPortalName() string {
	return fmt.Sprintf("DetailFieldPortal_%s", b.name)
}

func (b *DetailFieldBuilder) showElement(obj any, index int, ctx *web.EventContext) h.HTMLComponent {
	editBtn := VBtn("").Size(SizeXSmall).Variant("text").
		Rounded("0").
		Icon("mdi-square-edit-outline").
		Attr("v-if", fmt.Sprintf(`locals.showBtn`)).
		Attr("@click", web.Plaid().EventFunc(actions.DoEditDetailingListField).
			Query(DetailFieldName, b.name).
			Query(ParamID, ctx.Queries().Get(ParamID)).
			Query(b.EditBtnKey(), strconv.Itoa(index)).
			Go())

	return web.Portal(
		web.Scope(
			VCard(
				VCardText(
					h.Div(
						h.Div(b.elementShowFunc(obj, &FieldContext{
							Name:    b.name,
							FormKey: fmt.Sprintf("%s[%b]", b.name, index),
							Label:   b.label,
						}, ctx)).
							Class("flex-grow-1 pr-3"),
						h.Div(editBtn),
					).Class("d-flex justify-space-between"),
				),
			).Variant(VariantOutlined).
				Attr("@mouseenter", fmt.Sprintf(`locals.showBtn = true`)).
				Attr("@mouseleave", fmt.Sprintf(`locals.showBtn = false`)),
		).VSlot("{ locals }").Init(`{ showBtn:false  }`),
		h.Br(),
	).Name(b.ListElementPortalName(index))
}

func (b *DetailFieldBuilder) editElement(obj any, index, fromIndex int, ctx *web.EventContext) h.HTMLComponent {
	contentDiv := h.Div(
		h.Div(
			b.elementEditFunc(obj, &FieldContext{
				Name:    fmt.Sprintf("%s[%b]", b.name, index),
				FormKey: fmt.Sprintf("%s[%b]", b.name, index),
				Label:   fmt.Sprintf("%s[%b]", b.label, index),
			}, ctx),
		).Class("flex-grow-1"),
	).Class("d-flex justify-space-between")

	if !b.design.disableElementDeleteBtn {
		deleteBtn := VBtn("").Size(SizeXSmall).Variant("text").
			Rounded("0").
			Icon("mdi-delete-outline").
			Attr("@click", web.Plaid().EventFunc(actions.DoDeleteDetailingListField).
				Query(DetailFieldName, b.name).
				Query(ParamID, ctx.Queries().Get(ParamID)).
				Query(b.DeleteBtnKey(), index).
				Go())
		contentDiv.AppendChildren(h.Div(deleteBtn).Class("d-flex pl-3"))
	}

	saveBtn := VBtn("Save").Size(SizeSmall).
		Attr("style", "text-transform: none;").
		Attr("@click", web.Plaid().EventFunc(actions.DoSaveDetailingListField).
			Query(DetailFieldName, b.name).
			Query(ParamID, ctx.Queries().Get(ParamID)).
			Query(b.SaveBtnKey(), strconv.Itoa(index)).
			Go())

	card := VCard(
		VCardText(
			h.Div(
				contentDiv,
				h.Div(saveBtn).Class("ms-auto"),
			).Class("d-flex flex-column"),
		),
		h.Input("").Type("hidden").Attr(web.VField(b.ListElementIsEditing(index), true)...),
	).Variant(VariantOutlined).
		Attr("@mouseenter", fmt.Sprintf(`locals.showBtn = true`)).
		Attr("@mouseleave", fmt.Sprintf(`locals.showBtn = false`))

	return web.Portal(
		web.Scope(
			card,
		).VSlot("{ locals }").Init(`{ showBtn:false  }`),
		h.Br(),
	).Name(b.ListElementPortalName(index))
}

func (b *DetailFieldBuilder) DefaultElementUnmarshal(toObj, formObj any, prefix string, ctx *web.EventContext) error {
	if tf := reflect.TypeOf(toObj).Kind(); tf != reflect.Ptr {
		return errors.New(fmt.Sprintf("model %#+v must be pointer", toObj))
	}
	oldForm := &multipart.Form{
		Value: (map[string][]string)(http.Header(ctx.R.MultipartForm.Value).Clone()),
	}
	newForm := &multipart.Form{
		Value: make(map[string][]string),
	}
	// prefix.key => key
	for k, v := range oldForm.Value {
		if strings.HasPrefix(k, prefix+".") {
			newForm.Value[strings.TrimPrefix(k, prefix+".")] = v
		}
	}
	ctx2 := &web.EventContext{R: new(http.Request)}
	ctx2.R.MultipartForm = newForm

	_ = ctx2.UnmarshalForm(formObj)
	for _, f := range b.fields {
		name := f.name
		info := b.father.mb.modelInfo
		if info != nil {
			if info.Verifier().Do(PermCreate).ObjectOn(formObj).SnakeOn("f_"+name).WithReq(ctx.R).IsAllowed() != nil && info.Verifier().Do(PermUpdate).ObjectOn(formObj).SnakeOn("f_"+name).WithReq(ctx.R).IsAllowed() != nil {
				continue
			}
		}
		if v, err := reflectutils.Get(formObj, f.name); err == nil {
			reflectutils.Set(toObj, f.name, v)
		}
		if f.setterFunc == nil {
			continue
		}
		keyPath := fmt.Sprintf("%s.%s", prefix, f.name)
		err := f.setterFunc(toObj, &FieldContext{
			ModelInfo: info,
			FormKey:   keyPath,
			Name:      f.name,
			Label:     b.getLabel(f.NameLabel),
		}, ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
