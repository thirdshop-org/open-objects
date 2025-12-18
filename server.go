package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PartAPIResponse représente une pièce renvoyée par l'API
type PartAPIResponse struct {
	ID       int             `json:"id"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Props    json.RawMessage `json:"props"`
	Location string          `json:"location,omitempty"`
	Source   string          `json:"source,omitempty"` // "local" ou nom du peer
}

// LocationAPIResponse représente une localisation renvoyée par l'API
type LocationAPIResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ParentID    *int   `json:"parent_id,omitempty"`
	LocType     string `json:"loc_type"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// cmdServe lance un serveur HTTP
func cmdServe(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "Port HTTP")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// charger les templates HTML embarqués
	mustLoadWebTemplates()
	httpClient := &http.Client{Timeout: 500 * time.Millisecond}

	mux := http.NewServeMux()

	// asset statique htmx avec bon Content-Type
	mux.HandleFunc("/static/htmx.min.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		data, err := webFS.ReadFile("web/static/htmx.min.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})

	// qr-scanner (ESM) + worker
	mux.HandleFunc("/static/qr-scanner.min.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		data, err := webFS.ReadFile("web/static/qr-scanner.min.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})
	mux.HandleFunc("/static/qr-scanner-worker.min.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		data, err := webFS.ReadFile("web/static/qr-scanner-worker.min.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})

	// favicon.ico et source maps - retourne 404 silencieusement
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/static/qr-scanner.min.js.map", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/static/qr-scanner-worker.min.js.map", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// page d'accueil
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if err := tplIndex.ExecuteTemplate(w, "index", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// page de scan QR
	mux.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scan" {
			http.NotFound(w, r)
			return
		}
		if err := tplScan.ExecuteTemplate(w, "scan", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// page d'ajout de pièce
	mux.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/add" {
			http.NotFound(w, r)
			return
		}
		if err := tplAdd.ExecuteTemplate(w, "add", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// page localisation
	mux.HandleFunc("/location", func(w http.ResponseWriter, r *http.Request) {
		pathVal := r.URL.Query().Get("path")
		idVal := r.URL.Query().Get("id")
		if idVal == "" && pathVal == "" {
			http.Error(w, "id ou path manquant", http.StatusBadRequest)
			return
		}
		if pathVal == "" && idVal != "" {
			id, err := strconv.Atoi(idVal)
			if err != nil || id <= 0 {
				http.Error(w, "id invalide", http.StatusBadRequest)
				return
			}
			p, err := GetFullPath(db, id)
			if err != nil || p == "" {
				http.Error(w, "localisation introuvable", http.StatusNotFound)
				return
			}
			pathVal = p
		}
		data := struct {
			Path string
		}{Path: pathVal}
		if err := tplLocation.ExecuteTemplate(w, "location", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// page de détail
	mux.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Path[len("/view/"):]
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.NotFound(w, r)
			return
		}
		meta, err := GetPartMeta(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !meta.Found {
			http.NotFound(w, r)
			return
		}
		if err := tplView.ExecuteTemplate(w, "view", meta); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// partial recherche (htmx)
	mux.HandleFunc("/partials/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		typeName := ""
		nameSearch := q
		propSearch := ""
		// si la requête contient un ':' on le traite comme critère prop
		if strings.Contains(q, ":") {
			propSearch = q
			nameSearch = ""
		}
		results, err := searchParts(db, typeName, nameSearch, propSearch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data := struct {
			Results []PartAPIResponse
		}{Results: results}
		if err := tplSearch.ExecuteTemplate(w, "partials_search", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Récupération des champs de template: /api/template-fields?type=moteur
	mux.HandleFunc("/api/template-fields", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		typeName := r.URL.Query().Get("type")
		if typeName == "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{"fields": []interface{}{}})
			return
		}

		template, exists := Templates[typeName]
		if !exists {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}

		fields := []map[string]interface{}{}
		for fieldName, fieldDef := range template.Fields {
			field := map[string]interface{}{
				"name":        fieldName,
				"description": fieldDef.Description,
				"required":    fieldDef.Required,
				"type":        "text",
			}

			if fieldDef.Domain != "" {
				field["domain"] = fieldDef.Domain
				if fieldDef.Domain == "tension" || fieldDef.Domain == "puissance" || fieldDef.Domain == "vitesse_rot" || fieldDef.Domain == "dimension" {
					field["type"] = "number"
				}
			}

			if fieldDef.DefaultUnit != "" {
				field["unit"] = fieldDef.DefaultUnit
			}

			fields = append(fields, field)
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"fields": fields})
	})

	// Recherche: /api/search?type=...&name=...&prop=...
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		typeName := r.URL.Query().Get("type")
		nameSearch := r.URL.Query().Get("name")
		propSearch := r.URL.Query().Get("prop")

		results, err := searchParts(db, typeName, nameSearch, propSearch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Fan-out fédéré si aucun résultat local
		if len(results) == 0 {
			fed, _ := fetchFederated(db, httpClient, typeName, nameSearch, propSearch)
			results = append(results, fed...)
		}
		writeJSON(w, http.StatusOK, results)
	})

	// API fédérée (lecture seule) protégée par token
	mux.HandleFunc("/api/federated/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		token := extractBearer(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		if !isTokenAuthorized(db, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		typeName := r.URL.Query().Get("type")
		nameSearch := r.URL.Query().Get("name")
		propSearch := r.URL.Query().Get("prop")
		results, err := searchParts(db, typeName, nameSearch, propSearch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, results)
	})

	// Création de pièce: POST /api/parts (multipart form avec photos optionnelles)
	mux.HandleFunc("/api/parts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limiter la taille du formulaire (10MB)
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		// Parser le formulaire multipart
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "form too large or invalid", http.StatusBadRequest)
			return
		}

		// Récupérer les valeurs du formulaire
		payload := struct {
			Type  string
			Name  string
			Props map[string]interface{}
			Loc   string
		}{}

		payload.Type = r.FormValue("type")
		payload.Name = r.FormValue("name")
		payload.Loc = r.FormValue("loc")

		// Parser les propriétés JSON
		propsStr := r.FormValue("props")
		if propsStr != "" {
			if err := json.Unmarshal([]byte(propsStr), &payload.Props); err != nil {
				http.Error(w, "invalid props JSON", http.StatusBadRequest)
				return
			}
		} else {
			payload.Props = map[string]interface{}{}
		}

		// Validation de base
		if payload.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if payload.Type != "" && !TypeExists(payload.Type) {
			http.Error(w, fmt.Sprintf("type '%s' inconnu. Utilisez un template existant", payload.Type), http.StatusBadRequest)
			return
		}

		// Validation et normalisation des propriétés
		if payload.Type != "" {
			if err := ValidateProps(payload.Type, payload.Props); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		fieldUnits := GetFieldUnits(payload.Type)
		normProps, err := NormalizeProps(payload.Props, fieldUnits)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		propsJSON, err := json.Marshal(normProps)
		if err != nil {
			http.Error(w, "erreur sérialisation", http.StatusInternalServerError)
			return
		}

		// Gestion de la localisation
		var locationID *int
		if payload.Loc != "" {
			var id int
			if _, err := fmt.Sscanf(payload.Loc, "%d", &id); err == nil {
				if _, err := FindLocationByID(db, id); err != nil {
					http.Error(w, fmt.Sprintf("localisation ID %d inconnue", id), http.StatusBadRequest)
					return
				}
				locationID = &id
			} else {
				loc, err := FindLocationByName(db, payload.Loc)
				if err != nil {
					http.Error(w, fmt.Sprintf("localisation '%s' inconnue", payload.Loc), http.StatusBadRequest)
					return
				}
				locationID = &loc.ID
			}
		}

		// Créer la pièce
		id, err := CreatePart(db, payload.Type, payload.Name, string(propsJSON), locationID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Gestion des photos uploadées (optionnel)
		files := r.MultipartForm.File
		if len(files) > 0 {
			// Créer le dossier attachments s'il n'existe pas
			attachmentsDir := "attachments"
			if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
				log.Printf("Warning: cannot create attachments directory: %v", err)
			} else {
				// Sauvegarder chaque photo
				for _, fileHeaders := range files {
					for _, hdr := range fileHeaders {
						if strings.HasPrefix(hdr.Filename, "photo_") {
							file, err := hdr.Open()
							if err != nil {
								log.Printf("Warning: cannot open uploaded file %s: %v", hdr.Filename, err)
								continue
							}
							defer file.Close()

							// Générer un nom de fichier unique
							ext := filepath.Ext(hdr.Filename)
							if ext == "" {
								ext = ".jpg" // extension par défaut
							}
							filename := fmt.Sprintf("%d_%s%s", id, hdr.Filename, ext)
							filepath := filepath.Join(attachmentsDir, filename)

							// Sauvegarder le fichier
							dst, err := os.Create(filepath)
							if err != nil {
								log.Printf("Warning: cannot create file %s: %v", filepath, err)
								continue
							}
							defer dst.Close()

							if _, err := io.Copy(dst, file); err != nil {
								log.Printf("Warning: cannot save file %s: %v", filepath, err)
								continue
							}

							// Attacher le fichier à la pièce (utiliser la fonction existante)
							if _, err := AttachFile(db, int(id), filepath); err != nil {
								log.Printf("Warning: cannot attach file to part %d: %v", id, err)
							}
						}
					}
				}
			}
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"id":   id,
			"type": payload.Type,
			"name": payload.Name,
		})
	})

	// Localisations: GET /api/locations?search=...&id=...&path=...
	mux.HandleFunc("/api/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Recherche par ID
		if idStr := r.URL.Query().Get("id"); idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}
			loc, err := FindLocationByID(db, id)
			if err != nil {
				http.Error(w, "location not found", http.StatusNotFound)
				return
			}
			path, _ := GetFullPath(db, loc.ID)
			var pid *int
			if loc.ParentID.Valid {
				v := int(loc.ParentID.Int64)
				pid = &v
			}
			resp := LocationAPIResponse{
				ID:          loc.ID,
				Name:        loc.Name,
				ParentID:    pid,
				LocType:     loc.LocType,
				Description: loc.Description,
				Path:        path,
			}
			writeJSON(w, http.StatusOK, []LocationAPIResponse{resp})
			return
		}

		// Recherche par path
		if pathStr := r.URL.Query().Get("path"); pathStr != "" {
			limitStr := r.URL.Query().Get("limit")
			limit := 50 // limite par défaut
			if limitStr != "" {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
					limit = parsedLimit
				}
			}

			locs, err := ListLocations(db)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			var resp []LocationAPIResponse
			for _, l := range locs {
				path, _ := GetFullPath(db, l.ID)
				if strings.Contains(strings.ToLower(path), strings.ToLower(pathStr)) ||
				   strings.Contains(strings.ToLower(l.Name), strings.ToLower(pathStr)) {
					var pid *int
					if l.ParentID.Valid {
						v := int(l.ParentID.Int64)
						pid = &v
					}
					resp = append(resp, LocationAPIResponse{
						ID:          l.ID,
						Name:        l.Name,
						ParentID:    pid,
						LocType:     l.LocType,
						Description: l.Description,
						Path:        path,
					})

					// Respecter la limite
					if len(resp) >= limit {
						break
					}
				}
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}

		// Recherche textuelle
		if searchStr := r.URL.Query().Get("search"); searchStr != "" {
			limitStr := r.URL.Query().Get("limit")
			limit := 50 // limite par défaut
			if limitStr != "" {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
					limit = parsedLimit
				}
			}

			locs, err := ListLocations(db)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			var resp []LocationAPIResponse
			for _, l := range locs {
				path, _ := GetFullPath(db, l.ID)
				if strings.Contains(strings.ToLower(l.Name), strings.ToLower(searchStr)) ||
				   strings.Contains(strings.ToLower(path), strings.ToLower(searchStr)) {
					var pid *int
					if l.ParentID.Valid {
						v := int(l.ParentID.Int64)
						pid = &v
					}
					resp = append(resp, LocationAPIResponse{
						ID:          l.ID,
						Name:        l.Name,
						ParentID:    pid,
						LocType:     l.LocType,
						Description: l.Description,
						Path:        path,
					})

					// Respecter la limite
					if len(resp) >= limit {
						break
					}
				}
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}

		// Liste complète par défaut
		locs, err := ListLocations(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var resp []LocationAPIResponse
		for _, l := range locs {
			var pid *int
			if l.ParentID.Valid {
				v := int(l.ParentID.Int64)
				pid = &v
			}
			path, _ := GetFullPath(db, l.ID)
			resp = append(resp, LocationAPIResponse{
				ID:          l.ID,
				Name:        l.Name,
				ParentID:    pid,
				LocType:     l.LocType,
				Description: l.Description,
				Path:        path,
			})
		}
		writeJSON(w, http.StatusOK, resp)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("HTTP server listening on %s", addr)
	return http.ListenAndServe(addr, enableCORS(mux))
}

func searchParts(db *sql.DB, typeName, nameSearch, propSearch string) ([]PartAPIResponse, error) {
	criteria, err := MustCriteriaFromProp(propSearch)
	if err != nil {
		return nil, err
	}
	parts, err := SearchPartsDB(db, typeName, nameSearch, criteria)
	if err != nil {
		return nil, err
	}
	var results []PartAPIResponse
	for _, p := range parts {
		var locPath string
		if p.LocationID.Valid {
			locPath, _ = GetFullPath(db, int(p.LocationID.Int64))
		}
		propJSON := json.RawMessage("{}")
		if p.Props.Valid {
			propJSON = json.RawMessage(p.Props.String)
		}
		results = append(results, PartAPIResponse{
			ID:       p.ID,
			Type:     p.Type,
			Name:     p.Name,
			Props:    propJSON,
			Location: locPath,
			Source:   "local",
		})
	}
	return results, nil
}

// fetchFederated interroge les peers avec timeout et agrège les résultats
func fetchFederated(db *sql.DB, client *http.Client, typeName, nameSearch, propSearch string) ([]PartAPIResponse, error) {
	peers, err := ListPeers(db)
	if err != nil {
		return nil, err
	}
	if len(peers) == 0 {
		return nil, nil
	}

	type res struct {
		results []PartAPIResponse
	}
	ch := make(chan res, len(peers))

	for _, peer := range peers {
		p := peer
		go func() {
			url := fmt.Sprintf("%s/api/federated/search?type=%s&name=%s&prop=%s",
				strings.TrimRight(p.URL, "/"),
				urlQueryEscape(typeName),
				urlQueryEscape(nameSearch),
				urlQueryEscape(propSearch),
			)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				ch <- res{}
				return
			}
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
			resp, err := client.Do(req)
			if err != nil {
				ch <- res{}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				ch <- res{}
				return
			}
			var payload []PartAPIResponse
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				ch <- res{}
				return
			}
			for i := range payload {
				payload[i].Source = p.Name
			}
			ch <- res{results: payload}
		}()
	}

	var aggregated []PartAPIResponse
	for i := 0; i < len(peers); i++ {
		r := <-ch
		aggregated = append(aggregated, r.results...)
	}
	return aggregated, nil
}

func urlQueryEscape(s string) string {
	if s == "" {
		return ""
	}
	return template.URLQueryEscaper(s)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// enableCORS ajoute les headers CORS pour autoriser les requêtes externes
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractBearer récupère le token du header Authorization
func extractBearer(h string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(strings.TrimSpace(h), prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// isTokenAuthorized vérifie si un token correspond à un peer enregistré
func isTokenAuthorized(db *sql.DB, token string) bool {
	if token == "" {
		return false
	}
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM peers WHERE api_key = ?", token).Scan(&count)
	return count > 0
}
