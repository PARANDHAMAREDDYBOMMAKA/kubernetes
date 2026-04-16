package kaas

import (
	"net/http"

	"github.com/parandhamareddybommaka/kube/pkg/api/middleware"
	"github.com/parandhamareddybommaka/kube/pkg/auth"
)

// Mount installs the KaaS routes on the given mux.
// It wraps all handlers with Logger, Recovery, and CORS middleware, and
// additionally wraps the authenticated routes with auth.Required.
func (s *Server) Mount(mux *http.ServeMux) {
	common := []middleware.Middleware{middleware.Logger, middleware.Recovery, middleware.CORS}

	public := func(h http.HandlerFunc) http.Handler {
		return middleware.Chain(h, common...)
	}
	authed := func(h http.HandlerFunc) http.Handler {
		return middleware.Chain(auth.Required(h), common...)
	}

	// Public auth endpoints.
	mux.Handle("POST /api/v1/auth/register", public(s.Register))
	mux.Handle("POST /api/v1/auth/login", public(s.Login))
	mux.Handle("OPTIONS /api/v1/auth/register", public(okOptions))
	mux.Handle("OPTIONS /api/v1/auth/login", public(okOptions))

	// Authed auth endpoint.
	mux.Handle("GET /api/v1/auth/me", authed(s.Me))

	// Clusters.
	mux.Handle("GET /api/v1/clusters", authed(s.ListClusters))
	mux.Handle("POST /api/v1/clusters", authed(s.CreateCluster))
	mux.Handle("GET /api/v1/clusters/{id}", authed(s.GetCluster))
	mux.Handle("DELETE /api/v1/clusters/{id}", authed(s.DeleteCluster))
	mux.Handle("POST /api/v1/clusters/{id}/scale", authed(s.ScaleCluster))
	mux.Handle("GET /api/v1/clusters/{id}/kubeconfig", authed(s.Kubeconfig))
	mux.Handle("GET /api/v1/clusters/{id}/yaml", authed(s.ClusterYAML))

	// Load balancers.
	mux.Handle("GET /api/v1/loadbalancers", authed(s.ListLBs))
	mux.Handle("POST /api/v1/loadbalancers", authed(s.CreateLB))
	mux.Handle("GET /api/v1/loadbalancers/{id}", authed(s.GetLB))
	mux.Handle("DELETE /api/v1/loadbalancers/{id}", authed(s.DeleteLB))
	mux.Handle("GET /api/v1/loadbalancers/{id}/yaml", authed(s.LBYAML))
}

func okOptions(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
