package app

import (
	"context"
	"github.com/core-go/storage"
	"github.com/core-go/storage/google"
	"github.com/core-go/storage/s3"
	upload2 "go-service/internal/usescase/upload"
	"reflect"

	"github.com/core-go/auth"
	as "github.com/core-go/auth/sql"
	"github.com/core-go/code"
	. "github.com/core-go/health"
	"github.com/core-go/log/zap"
	. "github.com/core-go/security"
	. "github.com/core-go/security/jwt"
	. "github.com/core-go/security/sql"
	sv "github.com/core-go/service"
	"github.com/core-go/service/unique"
	v10 "github.com/core-go/service/v10"
	s "github.com/core-go/sql"
	_ "github.com/go-sql-driver/mysql"
)

type ApplicationContext struct {
	SkipSecurity          bool
	HealthHandler         *Handler
	AuthorizationHandler  *sv.AuthorizationHandler
	AuthorizationChecker  *AuthorizationChecker
	Authorizer            *Authorizer
	AuthenticationHandler *auth.AuthenticationHandler
	PrivilegesHandler     *auth.PrivilegesHandler
	CodeHandler           *code.Handler
	RolesHandler  		  *code.Handler
	UploadHandler 		  upload2.UploadHandler
	FileService   		  *upload2.FileService
}

func NewApp(ctx context.Context, conf Root) (*ApplicationContext, error) {
	db, er0 := s.Open(conf.DB)
	if er0 != nil {
		return nil, er0
	}
	sqlHealthChecker := s.NewHealthChecker(db)
	var healthHandler *Handler

	logError := log.ErrorMsg
	//generateId := shortid.Generate
	//var writeLog func(ctx context.Context, resource string, action string, success bool, desc string) error

	if conf.AuditLog.Log {
		auditLogDB, er1 := s.Open(conf.AuditLog.DB)
		if er1 != nil {
			return nil, er1
		}
		//logWriter := s.NewActionLogWriter(auditLogDB, "auditLog", conf.AuditLog.Config, conf.AuditLog.Schema, generateId)
		//writeLog = logWriter.Write
		auditLogHealthChecker := s.NewSqlHealthChecker(auditLogDB, "audit_log")
		healthHandler = NewHandler(sqlHealthChecker, auditLogHealthChecker)
	} else {
		healthHandler = NewHandler(sqlHealthChecker)
	}

	validator := v10.NewValidator()
	sqlPrivilegeLoader := NewPrivilegeLoader(db, conf.Sql.PermissionsByUser)

	userId := conf.Tracking.User
	tokenService := NewTokenService()
	authorizationHandler := sv.NewAuthorizationHandler(tokenService.GetAndVerifyToken, conf.Token.Secret)
	authorizationChecker := NewDefaultAuthorizationChecker(tokenService.GetAndVerifyToken, conf.Token.Secret, userId)
	authorizer := NewAuthorizer(sqlPrivilegeLoader.Privilege, true, userId)

	//authStatus := auth.InitStatus(conf.Status)
	//ldapAuthenticator, er2 := NewDAPAuthenticatorByConfig(conf.Ldap, authStatus)
	//if er2 != nil {
	//	return nil, er2
	//}
	//userInfoService, er3 := as.NewSqlUserInfoByConfig(db, conf.Auth)
	//if er3 != nil {
	//	return nil, er3
	//}
	//privilegeLoader, er4:= as.NewSqlPrivilegesLoader(db, conf.Sql.PrivilegesByUser, 1, true)
	//if er4 != nil {
	//	return nil, er4
	//}
	//authenticator := auth.NewBasicAuthenticator(authStatus, ldapAuthenticator.Authenticate, userInfoService, tokenService.GenerateToken, conf.Token, conf.Payload, privilegeLoader.Load)
	//authenticationHandler := auth.NewAuthenticationHandler(authenticator.Authenticate, authStatus.Error, authStatus.Timeout, logError, writeLog)

	privilegeReader, er5 := as.NewPrivilegesReader(db, conf.Sql.Privileges)
	if er5 != nil {
		return nil, er5
	}
	privilegeHandler := auth.NewPrivilegesHandler(privilegeReader.Privileges)

	// codeLoader := code.NewDynamicSqlCodeLoader(db, "select code, name, status as text from codeMaster where master = ? and status = 'A'", 1)
	codeLoader := code.NewSqlCodeLoader(db, "codeMaster", conf.Code.Loader)
	codeHandler := code.NewCodeHandlerByConfig(codeLoader.Load, conf.Code.Handler, logError)

	// rolesLoader := code.NewDynamicSqlCodeLoader(db, "select roleName as name, roleId as id, status as code from roles where status = 'A'", 0)
	rolesLoader := code.NewSqlCodeLoader(db, "roles", conf.Role.Loader)
	rolesHandler := code.NewCodeHandlerByConfig(rolesLoader.Load, conf.Role.Handler, logError)

	//roleService, er6 := r.NewRoleService(db, conf.Sql.Role.Check)
	//if er6 != nil {
	//	return nil, er6
	//}
	//roleValidator := unique.NewUniqueFieldValidator(db, "roles", "rolename", reflect.TypeOf(r.Role{}), validator.Validate)
	//generateRoleId := shortid.Func(conf.AutoRoleId)
	//roleHandler := r.NewRoleHandler(roleService, conf.Writer, logError, generateRoleId, roleValidator.Validate, conf.Tracking, writeLog)

	//userService, er7 := u.NewUserService(db)
	//if er7 != nil {
	//	return nil, er7
	//}
	//userValidator := unique.NewUniqueFieldValidator(db, "users", "username", reflect.TypeOf(u.User{}), validator.Validate)
	//generateUserId := shortid.Func(conf.AutoUserId)
	//userHandler := u.NewUserHandler(userService, conf.Writer, logError, generateUserId, userValidator.Validate, conf.Tracking, writeLog)

	storageService, er10 := CreateStorageService(ctx, conf)
	if er10 != nil {
		return nil, er10
	}
	//uploadType := reflect.TypeOf(upload2.Uploads{})
	//uploadRepository, er9 := s.NewRepository(db, "uploads", uploadType)
	//if er9 != nil {
	//	return nil, er9
	//}
	//uploadViewRepository, er11 := s.NewRepositoryWithArray(db, "uploads", uploadType, pq.Array)
	//if er11 != nil {
	//	return nil, er11
	//}
	uploadService, err11 := upload2.NewUploadService(db)
	if err11 != nil {
		return nil, err11
	}
	status := sv.InitializeStatus(conf.Status)
	action := sv.InitializeAction(conf.Action)
	uploadValidator := unique.NewUniqueFieldValidator(db, "uploads", "userid", reflect.TypeOf(upload2.Uploads{}), validator.Validate)
	uploadHandler := upload2.NewUploadHandler(uploadService, status, &action, logError, uploadValidator.Validate, db, storageService, conf.Storage.Directory, conf.KeyFile)

	//reportDB, er8 := s.Open(conf.AuditLog.DB)
	//if er8 != nil {
	//	return nil, er8
	//}
	//auditLogService, er9 := audit.NewAuditLogService(reportDB)
	//if er9 != nil {
	//	return nil, er9
	//}
	//auditLogHandler := audit.NewAuditLogHandler(auditLogService, logError, writeLog)

	// storage
	//storageService, er10 := CreateStorageService(ctx, conf)
	//if er10 != nil {
	//	return nil, er10
	//}
	//fileHandler := upload.NewCustomFileHandler(db, storageService, *uploadService, conf.Storage.Directory, conf.KeyFile, logError)

	app := &ApplicationContext{
		HealthHandler:         healthHandler,
		SkipSecurity:          conf.SecuritySkip,
		AuthorizationHandler:  authorizationHandler,
		AuthorizationChecker:  authorizationChecker,
		Authorizer:            authorizer,
		PrivilegesHandler:     privilegeHandler,
		CodeHandler:           codeHandler,
		RolesHandler:          rolesHandler,
		UploadHandler:         uploadHandler,
	}
	return app, nil
}

func CreateStorageService(ctx context.Context, root Root) (storage.StorageService, error) {
	if root.Provider == "google" {
		return google.NewGoogleStorageServiceWithCredentials(ctx, []byte(root.GoogleCredentials), root.Storage)
	} else {
		return s3.NewS3ServiceWithConfig(root.AWS, root.Storage)
	}
}
