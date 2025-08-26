package service

import (
	"go-cf-zone-switch/pkg/servers"
	"log"
)




type IPChanger struct {
	
}


func (r *IPChanger) ReportStatus(statuses []servers.ServerStatus) error {
	for _, s := range statuses {
		log.Printf("%+v\n", s)
	}
	return nil
}