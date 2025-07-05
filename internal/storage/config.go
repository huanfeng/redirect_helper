package storage

import (
	"redirect_helper/internal/config"
	"redirect_helper/internal/models"
)

type ConfigStorage struct {
	config *config.Config
}

func NewConfigStorage(cfg *config.Config) *ConfigStorage {
	return &ConfigStorage{
		config: cfg,
	}
}

func (s *ConfigStorage) CreateForwarding(name, token string) error {
	return s.config.AddForwarding(name, token)
}

func (s *ConfigStorage) SetTarget(name, token, target string) error {
	return s.config.SetTarget(name, token, target)
}

func (s *ConfigStorage) GetTarget(name string) (string, error) {
	return s.config.GetTarget(name)
}

func (s *ConfigStorage) GetForwarding(name string) (*models.ForwardingEntry, error) {
	forwarding, err := s.config.GetForwarding(name)
	if err != nil {
		return nil, err
	}

	return &models.ForwardingEntry{
		Name:      forwarding.Name,
		Token:     forwarding.Token,
		Target:    forwarding.Target,
		CreatedAt: forwarding.CreatedAt,
		UpdatedAt: forwarding.UpdatedAt,
	}, nil
}

func (s *ConfigStorage) ListForwardings() ([]*models.ForwardingEntry, error) {
	forwardings := s.config.ListForwardings()
	result := make([]*models.ForwardingEntry, 0, len(forwardings))

	for _, f := range forwardings {
		result = append(result, &models.ForwardingEntry{
			Name:      f.Name,
			Token:     f.Token,
			Target:    f.Target,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		})
	}

	return result, nil
}

func (s *ConfigStorage) RemoveForwarding(name string) error {
	return s.config.RemoveForwarding(name)
}

func (s *ConfigStorage) UpdateTarget(name, target string) error {
	return s.config.UpdateTarget(name, target)
}

// Domain methods implementation
func (s *ConfigStorage) CreateDomain(domain, token string) error {
	return s.config.AddDomain(domain, token)
}

func (s *ConfigStorage) SetDomainTarget(domain, token, target string) error {
	return s.config.SetDomainTarget(domain, token, target)
}

func (s *ConfigStorage) GetDomainTarget(domain string) (string, error) {
	return s.config.GetDomainTarget(domain)
}

func (s *ConfigStorage) GetDomain(domain string) (*models.DomainEntry, error) {
	domainConfig, err := s.config.GetDomain(domain)
	if err != nil {
		return nil, err
	}

	return &models.DomainEntry{
		Domain:    domainConfig.Domain,
		Token:     domainConfig.Token,
		Target:    domainConfig.Target,
		CreatedAt: domainConfig.CreatedAt,
		UpdatedAt: domainConfig.UpdatedAt,
	}, nil
}

func (s *ConfigStorage) ListDomains() ([]*models.DomainEntry, error) {
	domains := s.config.ListDomains()
	result := make([]*models.DomainEntry, 0, len(domains))

	for _, d := range domains {
		result = append(result, &models.DomainEntry{
			Domain:    d.Domain,
			Token:     d.Token,
			Target:    d.Target,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}

	return result, nil
}

func (s *ConfigStorage) RemoveDomain(domain string) error {
	return s.config.RemoveDomain(domain)
}

func (s *ConfigStorage) UpdateDomainTarget(domain, target string) error {
	return s.config.UpdateDomainTarget(domain, target)
}