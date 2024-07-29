package media

import (
	"github.com/qor5/web/v3"
)

const (
	openFileChooserEvent         = "mediaLibrary_OpenFileChooserEvent"
	deleteFileEvent              = "mediaLibrary_DeleteFileEvent"
	cropImageEvent               = "mediaLibrary_CropImageEvent"
	loadImageCropperEvent        = "mediaLibrary_LoadImageCropperEvent"
	imageSearchEvent             = "mediaLibrary_ImageSearchEvent"
	imageJumpPageEvent           = "mediaLibrary_ImageJumpPageEvent"
	uploadFileEvent              = "mediaLibrary_UploadFileEvent"
	chooseFileEvent              = "mediaLibrary_ChooseFileEvent"
	UpdateDescriptionEvent       = "mediaLibrary_UpdateDescriptionEvent"
	DeleteConfirmationEvent      = "mediaLibrary_DeleteConfirmationEvent"
	DoDeleteEvent                = "mediaLibrary_DoDelete"
	CreateFolderEvent            = "mediaLibrary_CreateFolderEvent"
	RenameDialogEvent            = "mediaLibrary_RenameDialogEvent"
	RenameEvent                  = "mediaLibrary_RenameEvent"
	UpdateDescriptionDialogEvent = "mediaLibrary_UpdateDescriptionDialogEvent"
	NewFolderDialogEvent         = "mediaLibrary_NewFolderDialogEvent"
	MoveToFolderDialogEvent      = "mediaLibrary_MoveToFolderDialogEvent"
	MoveToFolderEvent            = "mediaLibrary_MoveToFolderEvent"
	NextFolderEvent              = "mediaLibrary_NextFolderEvent"
)

func registerEventFuncs(hub web.EventFuncHub, mb *Builder) {
	hub.RegisterEventFunc(openFileChooserEvent, fileChooser(mb))
	hub.RegisterEventFunc(deleteFileEvent, deleteFileField())
	hub.RegisterEventFunc(cropImageEvent, cropImage(mb))
	hub.RegisterEventFunc(loadImageCropperEvent, loadImageCropper(mb))
	hub.RegisterEventFunc(imageSearchEvent, searchFile(mb))
	hub.RegisterEventFunc(imageJumpPageEvent, jumpPage(mb))
	hub.RegisterEventFunc(uploadFileEvent, uploadFile(mb))
	hub.RegisterEventFunc(chooseFileEvent, chooseFile(mb))
	hub.RegisterEventFunc(UpdateDescriptionEvent, updateDescription(mb))
	hub.RegisterEventFunc(DeleteConfirmationEvent, deleteConfirmation(mb))
	hub.RegisterEventFunc(DoDeleteEvent, doDelete(mb))
	hub.RegisterEventFunc(CreateFolderEvent, createFolder(mb))
	hub.RegisterEventFunc(NewFolderDialogEvent, newFolderDialog)
	hub.RegisterEventFunc(RenameDialogEvent, renameDialog(mb))
	hub.RegisterEventFunc(UpdateDescriptionDialogEvent, updateDescriptionDialog(mb))
	hub.RegisterEventFunc(RenameEvent, rename(mb))
	hub.RegisterEventFunc(MoveToFolderDialogEvent, moveToFolderDialog(mb))
	hub.RegisterEventFunc(MoveToFolderEvent, moveToFolder(mb))
	hub.RegisterEventFunc(NextFolderEvent, nextFolder(mb))
}
