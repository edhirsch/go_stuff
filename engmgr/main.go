package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// Engine pre-defined struct
// ------------------------------------
type Engine struct {
	Image      string   `json:"image"`
	Dockerfile string   `json:"dockerfile"`
	BuildHosts []string `json:"build-hosts"`
	Registries []string `json:"registry-hosts"`
	SyncHosts  []string `json:"sync-hosts"`
	Status     string
}

var engines []Engine

func getEngines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(engines)
}

func getEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	for _, item := range engines {
		if item.Image == params["engine"] {
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	json.NewEncoder(w).Encode(&Engine{})
}

func createEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var engine Engine
	_ = json.NewDecoder(r.Body).Decode(&engine)
	for _, item := range engines {
		if item.Image == engine.Image {
			w.WriteHeader(409)
			fmt.Fprintf(w, "Engine %s already exists\n", engine.Image)
			return
		}
	}
	engine.Status = "Created"
	engines = append(engines, engine)
	json.NewEncoder(w).Encode(&engine)
}

func buildEngine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range engines {
		if item.Image == params["engine"] {
			engines = append(engines[:index], engines[index+1:]...)
			var engine Engine
			_ = json.NewDecoder(r.Body).Decode(&engine)
			engine.Status = "Building image on build hosts"
			engines = append(engines, engine)
			json.NewEncoder(w).Encode(&engine)
			return
		}
	}
	json.NewEncoder(w).Encode(engines)
}

func pushEngine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range engines {
		if item.Image == params["engine"] {
			engines = append(engines[:index], engines[index+1:]...)
			var engine Engine
			_ = json.NewDecoder(r.Body).Decode(&engine)
			engine.Status = "Pushing image to the registry hosts"
			engines = append(engines, engine)
			json.NewEncoder(w).Encode(&engine)
			return
		}
	}
	json.NewEncoder(w).Encode(engines)
}

func syncEngine(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range engines {
		if item.Image == params["engine"] {
			engines = append(engines[:index], engines[index+1:]...)
			var engine Engine
			_ = json.NewDecoder(r.Body).Decode(&engine)
			engine.Status = "Synchronizing image to the sync hosts"
			engines = append(engines, engine)
			json.NewEncoder(w).Encode(&engine)
			return
		}
	}
	json.NewEncoder(w).Encode(engines)
}

func deleteEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	for index, item := range engines {
		if item.Image == params["engine"] {
			engines = append(engines[:index], engines[index+1:]...)
			fmt.Fprintf(w, "Engine %s deleted\n", params["engine"])
			break
		}
	}
	json.NewEncoder(w).Encode(engines)
}

func updateEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	for index, item := range engines {
		if item.Image == params["engine"] {
			engines = append(engines[:index], engines[index+1:]...)
			var engine Engine
			_ = json.NewDecoder(r.Body).Decode(&engine)
			engine.Status = "Updated"
			engines = append(engines, engine)
			json.NewEncoder(w).Encode(&engine)
			return
		}
	}
	json.NewEncoder(w).Encode(engines)
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/engines", getEngines).Methods("GET")
	router.HandleFunc("/engines/{engine}", getEngine).Methods("GET")
	router.HandleFunc("/create", createEngine).Methods("POST")
	router.HandleFunc("/build/{engine}", buildEngine).Methods("GET")
	router.HandleFunc("/push/{engine}", pushEngine).Methods("GET")
	router.HandleFunc("/sync/{engine}", syncEngine).Methods("GET")
	router.HandleFunc("/delete/{engine}", deleteEngine).Methods("GET")
	router.HandleFunc("/update/{engine}", updateEngine).Methods("POST")
	http.ListenAndServe("0.0.0.0:8000", router)
}
