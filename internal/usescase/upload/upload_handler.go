package upload

import (
	"bytes"
	"context"
	"database/sql"
	sv "github.com/core-go/service"
	"github.com/core-go/storage"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

type UploadHandler interface {
	All(w http.ResponseWriter, r *http.Request)
	Load(w http.ResponseWriter, r *http.Request)
	LoadImage(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
	UploadFile(w http.ResponseWriter, r *http.Request)
	DeleteFile(w http.ResponseWriter, r *http.Request)
}

type uploadHandler struct {
	service     UploadService
	fileService *FileService
	*sv.Params
}

func NewUploadHandler(
	uploadService UploadService,
	status sv.StatusConfig,
	action *sv.ActionConfig,
	logError func(context.Context, string),
	validate func(context.Context, interface{}) ([]sv.ErrorMessage, error),
	db *sql.DB, storageService storage.StorageService, directory string, KeyFile string) UploadHandler {
		modelType := reflect.TypeOf(Uploads{})
		params := sv.CreateParams(modelType, &status, logError, validate, action)
		fileService := NewCustomFileService(db, storageService, uploadService, directory, KeyFile)
		return &uploadHandler{uploadService, fileService, params}
}

func (h *uploadHandler) All(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.All(r.Context())
	sv.RespondModel(w, r, result, err,nil, nil)
}

func (h *uploadHandler) Load(w http.ResponseWriter, r *http.Request) {
	param := sv.GetParam(r)
	result, err := h.service.Load(r.Context(), param)
	sv.RespondModel(w, r, result, err,nil, nil)
}

func (h *uploadHandler) LoadImage(w http.ResponseWriter, r *http.Request) {
	param := sv.GetParam(r)
	urlEle := strings.Split(r.RequestURI, "/")
	checkImage, _ := findIndexInSlice(urlEle, "image")
	if checkImage >= 0 {
		result, err := h.service.LoadImage(r.Context(), param)
		sv.RespondModel(w, r, result, err, nil, nil)
	}
}

func (h *uploadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var upload Uploads
	er1 := sv.Decode(w, r, &upload)
	if er1 == nil {
		result, er3 := h.service.Create(r.Context(), &upload)
		sv.AfterCreated(w, r, &upload, result, er3, h.Status, h.Error, h.Log, h.Resource, h.Action.Create)
	}
}

func (h *uploadHandler) Update(w http.ResponseWriter, r *http.Request) {
	var upload Uploads
	er1 := sv.Decode(w, r, &upload)
	if er1 == nil {
		result, er3 := h.service.Update(r.Context(), &upload)
		sv.HandleResult(w, r, &upload, result, er3, h.Status, h.Error, h.Log, h.Resource, h.Action.Update)
	}
}

func (h *uploadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userId := r.FormValue("userId")
	if len(userId) <= 0 {
		http.Error(w, "No Id found!!!", http.StatusBadRequest)
		return
	}
	url := r.FormValue("url")
	if len(url) <= 0 {
		http.Error(w, "No url found!!!", http.StatusBadRequest)
		return
	}
	result, err := h.service.DeleteUpload(r.Context(), userId, url)
	sv.HandleDelete(w, r, result, err, h.Error, h.Log, h.Resource, h.Action.Delete)
}

func (h *uploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "not available", http.StatusInternalServerError)
		return
	}

	file, handler, err0 := r.FormFile("file")
	if err0 != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	bufferFile := bytes.NewBuffer(nil)
	if _, err1 := io.Copy(bufferFile, file); err1 != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	bytes := bufferFile.Bytes()
	contentType := handler.Header.Get("Content-Type")
	if len(contentType) == 0 {
		contentType = getExt(handler.Filename)
	}

	fileType := (strings.Split(contentType, "/"))[0]
	userId := r.FormValue("id")

	rs, err := h.fileService.UploadFile(r.Context(), "sub", userId, fileType, handler.Filename, bytes, contentType)
	if err != nil {
		panic(err)
	}
	sv.Succeed(w, r, http.StatusCreated, rs,nil, "", "")
}

func (h *uploadHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userId := r.FormValue("userId")
	url := ""
	if len(userId) <= 0 {
		http.Error(w, "require id", http.StatusBadRequest)
		return
	}
	r.ParseForm()
	url = r.FormValue("url")

	rs, err := h.fileService.DeleteFile(r.Context(), "sub", userId, url)
	if err != nil {
		panic(err)
	}
	sv.Succeed(w, r, http.StatusOK, rs, nil, "", "")
}

func findIndexInSlice(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func getExt(file string) string {
	ext := filepath.Ext(file)
	if strings.HasPrefix(ext, ":") {
		ext = ext[1:]
		return ext
	}
	return ext
}
