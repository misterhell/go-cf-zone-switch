package config

type At struct {
	Base          string `toml:"base"`
	DomainsTable  string `toml:"domains_table"`
	AccountsTable string `toml:"accounts_table"`
	AccountsView  string `toml:"accounts_view"`

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

type Config struct {
	At At `toml:"AT"`
}
