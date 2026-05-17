package cross_network_expose

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/netbirdio/netbird/management/server/account"
	nbcontext "github.com/netbirdio/netbird/management/server/context"
	"github.com/netbirdio/netbird/shared/management/http/util"
	"github.com/netbirdio/netbird/shared/management/proto"
	"github.com/netbirdio/netbird/shared/management/status"
)

type handler struct {
	accountManager account.Manager
	exposeManager  *account.CrossNetworkExposeManager
}

// AddEndpoints registers the cross-network expose HTTP endpoints
func AddEndpoints(accountManager account.Manager, router *mux.Router) {
	h := &handler{
		accountManager: accountManager,
		exposeManager:  account.NewCrossNetworkExposeManager(accountManager),
	}
	router.HandleFunc("/cross-network-exposes", h.listExposes).Methods("GET", "OPTIONS")
	router.HandleFunc("/cross-network-exposes", h.createExpose).Methods("POST", "OPTIONS")
	router.HandleFunc("/cross-network-exposes/{exposeId}", h.getExpose).Methods("GET", "OPTIONS")
	router.HandleFunc("/cross-network-exposes/{exposeId}", h.deleteExpose).Methods("DELETE", "OPTIONS")
}

// listExposes returns all cross-network exposes for the account
func (h *handler) listExposes(w http.ResponseWriter, r *http.Request) {
	userAuth, err := nbcontext.GetUserAuthFromContext(r.Context())
	if err != nil {
		log.WithContext(r.Context()).Error(err)
		http.Redirect(w, r, "/", http.StatusInternalServerError)
		return
	}

	exposes, err := h.exposeManager.ListCrossNetworkExposes(r.Context(), userAuth.AccountId)
	if err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	util.WriteJSONObject(r.Context(), w, exposes)
}

// createExpose creates a new cross-network expose
func (h *handler) createExpose(w http.ResponseWriter, r *http.Request) {
	userAuth, err := nbcontext.GetUserAuthFromContext(r.Context())
	if err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	var req struct {
		Name            string   `json:"name"`
		SourcePeerID    string   `json:"source_peer_id"`
		TargetAccountID string   `json:"target_account_id"`
		TargetNetworkID string   `json:"target_network_id"`
		TargetPeerID    string   `json:"target_peer_id"`
		Protocol        string   `json:"protocol"`
		Port            uint32   `json:"port"`
		Ports           []uint32 `json:"ports"`
		ListenPort      uint32   `json:"listen_port"`
		AllowedCIDRs    []string `json:"allowed_cidrs"`
		TTLHours        uint32   `json:"ttl_hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteErrorResponse("couldn't parse JSON request", http.StatusBadRequest, w)
		return
	}

	// Map protocol string to proto enum
	protocol := proto.ExposeProtocol_EXPOSE_TCP
	switch req.Protocol {
	case "udp", "UDP":
		protocol = proto.ExposeProtocol_EXPOSE_UDP
	case "http", "HTTP":
		protocol = proto.ExposeProtocol_EXPOSE_HTTP
	case "https", "HTTPS":
		protocol = proto.ExposeProtocol_EXPOSE_HTTPS
	case "tls", "TLS":
		protocol = proto.ExposeProtocol_EXPOSE_TLS
	}

	config := &account.CrossNetworkExposeConfig{
		Name:            req.Name,
		SourcePeerID:    req.SourcePeerID,
		TargetAccountID: req.TargetAccountID,
		TargetNetworkID: req.TargetNetworkID,
		TargetPeerID:    req.TargetPeerID,
		Protocol:        protocol,
		Port:            req.Port,
		Ports:           req.Ports,
		ListenPort:      req.ListenPort,
		AllowedCIDRs:    req.AllowedCIDRs,
		TTLHours:        req.TTLHours,
	}

	expose, err := h.exposeManager.CreateCrossNetworkExpose(r.Context(), userAuth.AccountId, userAuth.UserId, config)
	if err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	util.WriteJSONObject(r.Context(), w, expose)
}

// getExpose returns a single cross-network expose by ID
func (h *handler) getExpose(w http.ResponseWriter, r *http.Request) {
	userAuth, err := nbcontext.GetUserAuthFromContext(r.Context())
	if err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	vars := mux.Vars(r)
	exposeID, ok := vars["exposeId"]
	if !ok || exposeID == "" {
		util.WriteError(r.Context(), status.Errorf(status.InvalidArgument, "expose ID is missing"), w)
		return
	}

	expose, err := h.exposeManager.GetCrossNetworkExpose(r.Context(), userAuth.AccountId, exposeID)
	if err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	util.WriteJSONObject(r.Context(), w, expose)
}

// deleteExpose deletes a cross-network expose by ID
func (h *handler) deleteExpose(w http.ResponseWriter, r *http.Request) {
	userAuth, err := nbcontext.GetUserAuthFromContext(r.Context())
	if err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	vars := mux.Vars(r)
	exposeID, ok := vars["exposeId"]
	if !ok || exposeID == "" {
		util.WriteError(r.Context(), status.Errorf(status.InvalidArgument, "expose ID is missing"), w)
		return
	}

	if err := h.exposeManager.DeleteCrossNetworkExpose(r.Context(), userAuth.AccountId, exposeID); err != nil {
		util.WriteError(r.Context(), err, w)
		return
	}

	util.WriteJSONObject(r.Context(), w, emptyObject{})
}

type emptyObject struct{}
