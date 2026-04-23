package router

import (
	"net/http"

	"zookeeper/internal/controller"
)

type HTTPRouter struct {
	mux *http.ServeMux
}

func NewHTTPRouter(
	healthController *controller.HealthController,
	nodeController *controller.NodeController,
	electionController *controller.ElectionController,
) *HTTPRouter {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthController.GetHealth)
	mux.HandleFunc("/nodes/register", nodeController.RegisterNode)
	mux.HandleFunc("/nodes/heartbeat", nodeController.Heartbeat)
	mux.HandleFunc("/leader", nodeController.GetLeader)
	mux.HandleFunc("/nodes/alive", nodeController.GetAliveNodes)
	mux.HandleFunc("/election/state", electionController.GetElectionState)

	return &HTTPRouter{mux: mux}
}

func (r *HTTPRouter) Handler() http.Handler {
	return r.mux
}
