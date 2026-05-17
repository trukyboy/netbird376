package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func generateSubdomain() (string, error) {
	result := make([]byte, 6)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

func createDNSRecord(subdomain, ip, cfToken, zoneID string) error {
	payload := fmt.Sprintf(`{"type":"A","name":"%s.vp376.net","content":"%s","ttl":1,"proxied":false}`, subdomain, ip)
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID),
		nil)
	req.Header.Set("Authorization", "Bearer "+cfToken)
	req.Header.Set("Content-Type", "application/json")
	_ = payload
	// TODO: envoyer le payload
	return nil
}

func provisionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	subdomain, err := generateSubdomain()
	if err != nil {
		http.Error(w, "Failed to generate subdomain", 500)
		return
	}

	domain := subdomain + ".vp376.net"
	cfToken := os.Getenv("CF_TOKEN")
	zoneID := os.Getenv("CF_ZONE_ID")

	if err := createDNSRecord(subdomain, req.IP, cfToken, zoneID); err != nil {
		http.Error(w, "Failed to create DNS record", 500)
		return
	}

	log.Printf("Provisioned: %s -> %s", domain, req.IP)

	json.NewEncoder(w).Encode(map[string]string{
		"subdomain": subdomain,
		"domain":    domain,
		"token":     subdomain,
	})
}

func main() {
	http.HandleFunc("/api/provision", provisionHandler)
	log.Println("Provisioner listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
