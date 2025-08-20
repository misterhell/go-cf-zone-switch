package at

type AtConfig interface {
	GetBase() string
	GetDomainsTable() string
	GetAccountTable() string
	GetAccountView() string
	GetApiToken() string
	GetHostingTable() string
}