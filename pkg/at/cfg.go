package at

type AtConfig interface {
	GetBase() string
	GetDomainsTable() string
	GetHostingTable() string
	GetAccountTable() string
	GetAccountView() string
	GetApiToken() string
}
