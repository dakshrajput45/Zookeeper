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
	replicationController *controller.ReplicationController,
) *HTTPRouter {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthController.GetHealth)
	mux.HandleFunc("/nodes/register", nodeController.RegisterNode)
	mux.HandleFunc("/nodes/heartbeat", nodeController.Heartbeat)
	mux.HandleFunc("/leader", nodeController.GetLeader)
	mux.HandleFunc("/nodes/alive", nodeController.GetAliveNodes)
	mux.HandleFunc("/election/state", electionController.GetElectionState)
	mux.HandleFunc("/write", replicationController.Write)
	mux.HandleFunc("/read", replicationController.Read)
	mux.HandleFunc("/replication/state", replicationController.GetState)

	return &HTTPRouter{mux: mux}
}

func (r *HTTPRouter) Handler() http.Handler {
	return r.mux
}
