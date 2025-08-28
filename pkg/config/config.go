package config

type At struct {
	Base             string `toml:"base"`
	DomainsTable     string `toml:"domains_table"`
	AccountsTable    string `toml:"accounts_table"`
	AccountsView     string `toml:"accounts_view"`
	HostingTable     string `toml:"hosting_table"`
	DomainsUpdateMin int    `toml:"domains_update_min"`

	Token string `toml:"token"`
}

func (a At) GetBase() string {
	return a.Base
}

func (a At) GetAccountTable() string {
	return a.AccountsTable
}

func (a At) GetAccountView() string {
	return a.AccountsView
}

func (a At) GetDomainsTable() string {
	return a.DomainsTable
}

func (a At) GetApiToken() string {
	return a.Token
}

func (a At) GetHostingTable() string {
	return a.HostingTable
}

type Servers struct {
	Proxy            []string `toml:"proxy"`
	CheckIntervalSec int      `toml:"check_interval_sec"`
	TimeoutSec       int      `toml:"timeout_sec"`

	ProxyConfEndpoint          string `toml:"proxy_conf_endpoint"`
	ProxyConfUpdateINtervalMin int    `toml:"proxy_conf_update_interval_min"`
	DomainUpdateEndpoint       string `toml:"domain_update_endpoint"`
	DomainUpdateIntervalMin    int    `toml:"domain_update_interval_min"`
}

type Config struct {
	At      At      `toml:"AT"`
	Servers Servers `toml:"Servers"`
}
