package main


import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type Record struct {
	Domain string `json:"domain"`
	Ip     string `json:"ip"`
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var records []Record
	if err := json.Unmarshal(body, &records); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the total number of elements received.
	log.Printf("Received %d records", len(records))
	log.Printf("%+v", records[len(records)- 1])
	// Get the last 5 elements (if available).
	start := 0
	if len(records) > 5 {
		start = len(records) - 5
	}
	lastFive := records[start:]

	// Prepare response.
	response := struct {
		Count      int      `json:"count"`
		LastRecord []Record `json:"last_record"`
	}{
		Count:      len(records),
		LastRecord: lastFive,
	}

	res, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error generating output", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func main() {
	http.HandleFunc("/update", updateHandler)
	log.Println("Server started on port 8686")
	http.ListenAndServe(":8686", nil)
}