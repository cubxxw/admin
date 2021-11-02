package views

import (
	"fmt"

	"github.com/goplaid/web"
	"github.com/goplaid/x/i18n"
	"github.com/goplaid/x/presets"
	. "github.com/goplaid/x/vuetify"
	"github.com/qor/qor5/publish"
	"github.com/qor/qor5/utils"
	h "github.com/theplant/htmlgo"
	"github.com/theplant/jsontyperegistry"
)

func StatusListFunc() presets.FieldComponentFunc {
	return func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)

		if s, ok := obj.(publish.StatusInterface); ok {
			return h.Td(VChip(h.Text(GetStatusText(s.GetStatus(), msgr))).Color(getStatusColor(s.GetStatus())))
		}
		return nil
	}
}

func StatusEditFunc() presets.FieldComponentFunc {
	return func(obj interface{}, field *presets.FieldContext, ctx *web.EventContext) h.HTMLComponent {
		s, ok := obj.(publish.StatusInterface)
		if !ok || s.GetStatus() == "" {
			return nil
		}

		msgr := i18n.MustGetModuleMessages(ctx.R, I18nPublishKey, Messages_en_US).(*Messages)
		utilsMsgr := i18n.MustGetModuleMessages(ctx.R, utils.I18nUtilsKey, Messages_en_US).(*utils.Messages)

		var btn h.HTMLComponent
		switch s.GetStatus() {
		case publish.StatusDraft, publish.StatusOffline:
			btn = VBtn(msgr.Publish).Attr("@click", fmt.Sprintf(`locals.action="%s";locals.commonConfirmDialog = true`, publishEvent))
		case publish.StatusOnline:
			btn = h.Div(
				VBtn(msgr.Unpublish).Attr("@click", fmt.Sprintf(`locals.action="%s";locals.commonConfirmDialog = true`, unpublishEvent)),
				VBtn(msgr.Republish).Attr("@click", fmt.Sprintf(`locals.action="%s";locals.commonConfirmDialog = true`, publishEvent)),
			)
		}

		//params := []interface{}{reflect.TypeOf(obj).String(), fmt.Sprint(reflectutils.MustGet(obj, "ID"))}
		//if v, ok := obj.(publish.VersionInterface); ok {
		//	params = append(params, v.GetVersionName())
		//}

		return h.Div(
			VStepper(
				VStepperHeader(
					VStepperStep(h.Text(msgr.StatusDraft)).Step(0).Complete(s.GetStatus() == publish.StatusDraft),
					VDivider(),
					VStepperStep(h.Text(msgr.StatusOnline)).Step(0).Complete(s.GetStatus() == publish.StatusOnline),
				),
			),
			h.Br(),
			btn,
			utils.ConfirmDialog(msgr.Areyousure, web.Plaid().EventFunc(web.Var("locals.action")).
				FieldValue("objJson", jsontyperegistry.MustJSONString(obj)).Go(), utilsMsgr)).
			Attr(web.InitContextLocals, `{ action: ""}`)

	}
}

func getStatusColor(status string) string {
	switch status {
	case publish.StatusDraft:
		return "orange"
	case publish.StatusOnline:
		return "green"
	case publish.StatusOffline:
		return "grey"
	}
	return ""
}
