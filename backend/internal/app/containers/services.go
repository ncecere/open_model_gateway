package containers

import (
	"time"

	"github.com/ncecere/open_model_gateway/backend/internal/accounts"
	"github.com/ncecere/open_model_gateway/backend/internal/auth"
	"github.com/ncecere/open_model_gateway/backend/internal/catalog"
	adminapikeyssvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminapikeys"
	adminauditsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminaudit"
	adminbudgetsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminbudget"
	admincatalogsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admincatalog"
	adminconfigsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminconfig"
	adminprovidersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminprovider"
	adminratelimitsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminratelimit"
	adminrbacsvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminrbac"
	admintenantsvc "github.com/ncecere/open_model_gateway/backend/internal/services/admintenant"
	adminusersvc "github.com/ncecere/open_model_gateway/backend/internal/services/adminuser"
	batchsvc "github.com/ncecere/open_model_gateway/backend/internal/services/batches"
	billinghooks "github.com/ncecere/open_model_gateway/backend/internal/services/billinghooks"
	exportsvc "github.com/ncecere/open_model_gateway/backend/internal/services/exports"
	filesvc "github.com/ncecere/open_model_gateway/backend/internal/services/files"
	tenantservice "github.com/ncecere/open_model_gateway/backend/internal/services/tenant"
	usageservice "github.com/ncecere/open_model_gateway/backend/internal/services/usage"
)

// ServiceContainer holds business logic services.
type ServiceContainer struct {
	// Account services
	Accounts      *accounts.PersonalService
	AdminUsers    *adminusersvc.Service
	AdminAuth     *auth.AdminAuthService
	AdminRBAC     *adminrbacsvc.Service
	TenantService *tenantservice.Service

	// Admin management services
	AdminTenants    *admintenantsvc.Service
	AdminCatalog    *admincatalogsvc.Service
	AdminBudgets    *adminbudgetsvc.Service
	AdminRateLimits *adminratelimitsvc.Service
	AdminProviders  *adminprovidersvc.Service
	AdminConfig     *adminconfigsvc.Service
	AdminAudit      *adminauditsvc.Service
	AdminAPIKeys    *adminapikeyssvc.Service

	// Resource services
	DefaultModels   *catalog.DefaultModelService
	UsageService    *usageservice.Service
	Batches         *batchsvc.Service
	Files           *filesvc.Service
	Exports         *exportsvc.Service
	BillingWebhooks *billinghooks.Service

	// Reporting location for timezone-aware operations
	ReportingLocation *time.Location
}
