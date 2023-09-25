package seo

type Messages struct {
	PageTitle               string
	Variable                string
	VariableDescription     string
	PageMetadataTitle       string
	PageMetadataDescription string
	Basic                   string
	Title                   string
	Description             string
	Keywords                string
	OpenGraphInformation    string
	OpenGraphTitle          string
	OpenGraphDescription    string
	OpenGraphURL            string
	OpenGraphType           string
	OpenGraphImageURL       string
	OpenGraphImage          string
	OpenGraphMetadata       string
	Save                    string
	SavedSuccessfully       string
	Seo                     string
	UseDefaults             string
	GlobalName              string
}

var Messages_en_US = &Messages{
	PageTitle:               "SEO Setting",
	Variable:                "Variables Setting",
	PageMetadataTitle:       "Page Metadata Defaults",
	PageMetadataDescription: "These defaults are for pages automatically generated by the system, you can override them on the individual pages.",
	Basic:                   "Basic",
	Title:                   "Title",
	Description:             "Description",
	Keywords:                "Keywords",
	OpenGraphInformation:    "Open Graph Information",
	OpenGraphTitle:          "Open Graph Title",
	OpenGraphDescription:    "Open Graph Description",
	OpenGraphURL:            "Open Graph URL",
	OpenGraphType:           "Open Graph Type",
	OpenGraphImageURL:       "Open Graph Image URL",
	OpenGraphImage:          "Open Graph Image",
	OpenGraphMetadata:       "Open Graph Metadata",
	Save:                    "Save",
	SavedSuccessfully:       "Saved successfully",
	Seo:                     "SEO",
	UseDefaults:             "Use Defaults",
	GlobalName:              "Global Default SEO",
}

var Messages_zh_CN = &Messages{
	PageTitle:               "搜索引擎优化设置",
	Variable:                "变量设置",
	PageMetadataTitle:       "页面默认值",
	PageMetadataDescription: "这些默认值适用于系统自动生成的页面，您可以在各个页面上覆盖它们。",
	Basic:                   "基本信息",
	Title:                   "标题",
	Description:             "描述",
	Keywords:                "关键词",
	OpenGraphInformation:    "OG 信息",
	OpenGraphTitle:          "OG 标题",
	OpenGraphDescription:    "OG 描述",
	OpenGraphURL:            "OG 链接",
	OpenGraphType:           "OG 类型",
	OpenGraphImageURL:       "OG 图片链接",
	OpenGraphImage:          "OG 图片",
	OpenGraphMetadata:       "OG 元数据",
	Save:                    "保存",
	SavedSuccessfully:       "成功保存",
	Seo:                     "搜索引擎优化",
	UseDefaults:             "使用默认值",
	GlobalName:              "全局默认SEO",
}
