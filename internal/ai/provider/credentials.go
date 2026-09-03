package provider

type credentialStore interface {
	Save(id, secret string) error
	Load(id string) (string, error)
}
