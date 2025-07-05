package storage

import "redirect_helper/internal/models"

type Storage interface {
	CreateForwarding(name, token string) error
	SetTarget(name, token, target string) error
	GetTarget(name string) (string, error)
	GetForwarding(name string) (*models.ForwardingEntry, error)
	ListForwardings() ([]*models.ForwardingEntry, error)
}

type ExtendedStorage interface {
	Storage
	RemoveForwarding(name string) error
	UpdateTarget(name, target string) error
}

type DomainStorage interface {
	CreateDomain(domain, token string) error
	SetDomainTarget(domain, token, target string) error
	GetDomainTarget(domain string) (string, error)
	GetDomain(domain string) (*models.DomainEntry, error)
	ListDomains() ([]*models.DomainEntry, error)
	RemoveDomain(domain string) error
	UpdateDomainTarget(domain, target string) error
}