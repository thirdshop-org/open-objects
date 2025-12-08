package main

import (
	"database/sql"
)

type Peer struct {
	ID    int
	Name  string
	URL   string
	APIKey string
}

func AddPeer(db *sql.DB, name, url, token string) error {
	_, err := db.Exec(`INSERT INTO peers (name, url, api_key) VALUES (?, ?, ?)`, name, url, token)
	return err
}

func ListPeers(db *sql.DB) ([]Peer, error) {
	rows, err := db.Query(`SELECT id, name, url, api_key FROM peers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var peers []Peer
	for rows.Next() {
		var p Peer
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.APIKey); err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, nil
}

func GetPeerByID(db *sql.DB, id int) (*Peer, error) {
	var p Peer
	err := db.QueryRow(`SELECT id, name, url, api_key FROM peers WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.URL, &p.APIKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

