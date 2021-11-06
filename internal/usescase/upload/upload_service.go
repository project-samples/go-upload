package upload

import (
	"context"
	"database/sql"
	"fmt"
	q "github.com/core-go/sql"
	"github.com/core-go/storage"
	"github.com/lib/pq"
	"reflect"
	"strings"
)

type UploadService interface {
	All(ctx context.Context) ([]Uploads, error)
	Load(ctx context.Context, id string) ([]FileUploads, error)
	LoadImage(ctx context.Context, id string) ([]string, error)
	Create(ctx context.Context, upload *Uploads) (int64, error)
	Update(ctx context.Context, upload *Uploads) (int64, error)
	DeleteUpload(ctx context.Context, id string, url string) (int64, error)
}

type uploadService struct {
	db *sql.DB
	BuildParam   func(int) string
	CheckDelete  string
	Map          map[string]int
	modelType    reflect.Type
	uploadSchema *q.Schema
}

type FileService struct {
	StorageService storage.StorageService
	UploadService  UploadService
	Directory      string
	KeyFile        string
	Error          func(context.Context, string)
	db             *sql.DB
}

func NewCustomFileService(db *sql.DB, storageService storage.StorageService, uploadService UploadService, directory string, keyFile string, options ...func(context.Context, string)) *FileService {
	var logError func(context.Context, string)
	if len(options) > 0 && options[0] != nil {
		logError = options[0]
	}
	return &FileService{db: db, StorageService: storageService, UploadService: uploadService, Directory: directory, KeyFile: keyFile, Error: logError}
}

func NewUploadService(db *sql.DB) (UploadService, error) {
	var model Uploads
	var subModel FileUploads
	modelType := reflect.TypeOf(model)
	buildParam := q.GetBuild(db)
	subType := reflect.TypeOf(subModel)
	m, err := q.GetColumnIndexes(subType)
	if err != nil {
		return nil, err
	}
	uploadSchema := q.CreateSchema(modelType)
	return &uploadService{db: db, BuildParam: buildParam, modelType: modelType, Map: m, uploadSchema: uploadSchema}, nil
}

func (s *uploadService) All(ctx context.Context) ([]Uploads, error) {
	var uploadRes []Uploads
	query := "select * from uploads"
	res, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	uploadRes, err = uploadResult(res)
	return uploadRes, nil
}

func (s *uploadService) Load(ctx context.Context, id string) ([]FileUploads, error) {
	query := fmt.Sprintf("select * from uploads where userid = '%s'", id)
	res, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	uploadRes, err := uploadResult(res)
	if err != nil {
		return nil, err
	}
	return uploadRes[0].Data, nil
}

func (s *uploadService) LoadImage(ctx context.Context, id string) ([]string, error) {
	rs, err := s.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	var resultListImage []string
	for _, v := range rs {
		resultListImage = append(resultListImage, v.Url)
	}
	return resultListImage, nil
}

func (s *uploadService) Create(ctx context.Context, upload *Uploads) (int64, error) {
	statement, err := BuildCreateUploadStatement(upload, s.BuildParam, s.uploadSchema)
	if err != nil {
		return 0, err
	}
	return statement.Exec(ctx, s.db)
}

func (s *uploadService) Update(ctx context.Context, upload *Uploads) (int64, error) {
	statement, err := BuildUpdateUploadStatement(upload, s.BuildParam, s.uploadSchema)
	if err != nil {
		return 0, err
	}
	return statement.Exec(ctx, s.db)
}

func (s *uploadService) DeleteUpload(ctx context.Context, id string, url string) (int64, error) {
	rs, err := s.Load(ctx, id)
	if err != nil {
		return -1, err
	}

	var finalResultData []FileUploads
	for i, v := range rs {
		if url == v.Url {
			finalResultData = removeElementInSlice(rs, i)
		}
	}
	if len(finalResultData) == 0 {
		rs = []FileUploads{}
	} else {
		rs = finalResultData
	}

	var resUploadData = &Uploads{
		UserId: id,
		Data: rs,
	}

	statement, err := BuildUpdateUploadStatement(resUploadData, s.BuildParam, s.uploadSchema)
	if err != nil {
		return 0, err
	}
	resQuery, err := statement.Exec(ctx, s.db)
	if err != nil {
		return 0, err
	}
	return resQuery, nil
}

func (f *FileService) UploadFile(ctx context.Context, directory string, id string, fileType string, fileName string, bytes []byte, contentType string) (string, error) {
	rs, err2 := f.StorageService.Upload(ctx, directory, fileName, bytes, contentType)
	if err2 != nil {
		return "cannot upload file", err2
	}

	// database
	var newFile = FileUploads{
		Url:      rs,
		Source:   "google-storage",
		Type: fileType,
	}

	resUpload, err3 := f.UploadService.Load(ctx, id)
	if err3 != nil {
		panic(err3)
	}

	resUpload = append(resUpload, newFile)

	var resUploadData = &Uploads{
		UserId: id,
		Data: resUpload,
	}

	_, err4 := f.UploadService.Update(ctx, resUploadData)
	if err4 != nil {
		panic(err4)
	}

	return rs, nil
}

func (f *FileService) DeleteFile(ctx context.Context, directory string, id string, url string) (bool, error) {
	fileName := getFileNameInURL(url)
	rs, err2 := f.StorageService.Delete(ctx, directory, fileName)
	if err2 != nil && rs == false {
		return rs, err2
	}

	// database
	resUpload, err3 := f.UploadService.Load(ctx, id)
	if err3 != nil {
		panic(err3)
	}

	var finalResultData []FileUploads
	for i, v := range resUpload {
		fileNameUrl := getFileNameInURL(v.Url)
		if fileNameUrl == fileName {
			finalResultData = removeElementInSlice(resUpload, i)
		}
	}
	resUpload = finalResultData

	var resUploadData = &Uploads{
		UserId: id,
		Data: resUpload,
	}

	_, err4 := f.UploadService.Update(ctx, resUploadData)
	if err4 != nil {
		panic(err4)
	}

	return rs, nil
}

func uploadResult(query *sql.Rows) ([]Uploads, error) {
	var res []Uploads
	upload := Uploads{}
	for query.Next() {
		err := query.Scan(&upload.UserId, pq.Array(&upload.Data))
		if err != nil {
			return nil, err
		}
		res = append(res, upload)
	}
	return res, nil
}

func BuildCreateUploadStatement(obj interface{}, buildParam func(int) string, uploadSchema *q.Schema) (q.Statements, error) {
	_, ok := obj.(*Uploads)
	if !ok {
		return nil, fmt.Errorf("invalid obj model from request")
	}
	sts := q.NewStatements(true)
	sts.Add(q.BuildToInsertWithArray("uploads", obj, buildParam, true, pq.Array, uploadSchema))
	return sts, nil
}

func BuildUpdateUploadStatement(obj interface{}, buildParam func(int) string, uploadSchema *q.Schema) (q.Statements, error) {
	_, ok := obj.(*Uploads)
	if !ok {
		return nil, fmt.Errorf("invalid obj model from request")
	}
	sts := q.NewStatements(true)
	sts.Add(q.BuildToUpdateWithArray("uploads", obj, buildParam, true, pq.Array, uploadSchema))
	return sts, nil
}

func removeElementInSlice(slice []FileUploads, s int) []FileUploads {
	return append(slice[:s], slice[s+1:]...)
}

func getFileNameInURL(url string) string {
	fileNameElement := strings.Split(url, "/")
	fileNameUrl := fileNameElement[len(fileNameElement)-1]
	return fileNameUrl
}
