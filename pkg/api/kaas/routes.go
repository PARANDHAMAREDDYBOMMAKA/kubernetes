package kaas

import (
	"net/http"

	"github.com/parandhamareddybommaka/kube/pkg/api/middleware"
	"github.com/parandhamareddybommaka/kube/pkg/auth"
)

func (s *Server) Mount(mux *http.ServeMux) {
	common := []middleware.Middleware{middleware.Logger, middleware.Recovery, middleware.CORS}
	authRateLimit := middleware.PerIPRateLimit(10, 10)

	public := func(h http.HandlerFunc) http.Handler {
		return middleware.Chain(h, common...)
	}
	rateLimited := func(h http.HandlerFunc) http.Handler {
		return middleware.Chain(h, append([]middleware.Middleware{authRateLimit}, common...)...)
	}
	authed := func(h http.HandlerFunc) http.Handler {
		return middleware.Chain(auth.Required(h), common...)
	}

	mux.Handle("POST /api/v1/auth/register", rateLimited(s.Register))
	mux.Handle("POST /api/v1/auth/login", rateLimited(s.Login))
	mux.Handle("OPTIONS /api/v1/auth/register", public(okOptions))
	mux.Handle("OPTIONS /api/v1/auth/login", public(okOptions))

	mux.Handle("GET /api/v1/auth/me", authed(s.Me))

	mux.Handle("GET /api/v1/clusters", authed(s.ListClusters))
	mux.Handle("POST /api/v1/clusters", authed(s.CreateCluster))
	mux.Handle("GET /api/v1/clusters/{id}", authed(s.GetCluster))
	mux.Handle("DELETE /api/v1/clusters/{id}", authed(s.DeleteCluster))
	mux.Handle("POST /api/v1/clusters/{id}/scale", authed(s.ScaleCluster))
	mux.Handle("GET /api/v1/clusters/{id}/kubeconfig", authed(s.Kubeconfig))
	mux.Handle("GET /api/v1/clusters/{id}/yaml", authed(s.ClusterYAML))
	mux.Handle("GET /api/v1/clusters/{id}/nodes", authed(s.ListNodes))

	mux.Handle("GET /api/v1/loadbalancers", authed(s.ListLBs))
	mux.Handle("POST /api/v1/loadbalancers", authed(s.CreateLB))
	mux.Handle("GET /api/v1/loadbalancers/{id}", authed(s.GetLB))
	mux.Handle("DELETE /api/v1/loadbalancers/{id}", authed(s.DeleteLB))
	mux.Handle("GET /api/v1/loadbalancers/{id}/yaml", authed(s.LBYAML))
}

func okOptions(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
