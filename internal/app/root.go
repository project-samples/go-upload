package app

import (
	. "github.com/core-go/auth"
	. "github.com/core-go/auth/ldap"
	as "github.com/core-go/auth/sql"
	"github.com/core-go/code"
	mid "github.com/core-go/log/middleware"
	"github.com/core-go/log/zap"
	sv "github.com/core-go/service"
	"github.com/core-go/service/audit"
	. "github.com/core-go/service/model-builder"
	"github.com/core-go/sql"
	"github.com/core-go/storage"
	"github.com/core-go/storage/s3"
)

type Root struct {
	Status            *sv.StatusConfig 	   `mapstructure:"status"`
	Action    		  *sv.ActionConfig 	   `mapstructure:"action"`
	Server            sv.ServerConf        `mapstructure:"server"`
	SecuritySkip      bool                 `mapstructure:"security_skip"`
	Ldap              LDAPConfig           `mapstructure:"ldap"`
	Auth              as.SqlConfig         `mapstructure:"auth"`
	Token             TokenConfig          `mapstructure:"token"`
	Payload           PayloadConfig        `mapstructure:"payload"`
	DB                sql.Config           `mapstructure:"db"`
	Log               log.Config           `mapstructure:"log"`
	MiddleWare        mid.LogConfig        `mapstructure:"middleware"`
	AutoRoleId        *bool                `mapstructure:"auto_role_id"`
	AutoUserId        *bool                `mapstructure:"auto_user_id"`
	Role              code.Config          `mapstructure:"role"`
	Upload            code.Config          `mapstructure:"upload"`
	Code              code.Config          `mapstructure:"code"`
	AuditLog          sql.ActionLogConf    `mapstructure:"audit_log"`
	AuditClient       audit.AuditLogClient `mapstructure:"audit_client"`
	Writer            sv.WriterConfig      `mapstructure:"writer"`
	Tracking          TrackingConfig       `mapstructure:"tracking"`
	Sql               SqlStatement         `mapstructure:"sql"`
	Provider          string               `mapstructure:"provider"`
	GoogleCredentials string               `mapstructure:"google_credentials"`
	AWS               s3.Config            `mapstructure:"aws"`
	Storage           storage.Config       `mapstructure:"storage"`
	KeyFile           string               `mapstructure:"key_file"`
}

type SqlStatement struct {
	Privileges        string        `mapstructure:"privileges"`
	PrivilegesByUser  string        `mapstructure:"privileges_by_user"`
	PermissionsByUser string        `mapstructure:"permissions_by_user"`
	Role              RoleStatement `mapstructure:"role"`
}

type RoleStatement struct {
	Check string `mapstructure:"check"`
}
