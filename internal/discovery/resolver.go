package discovery

import (
	"errors"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

// ErrDiscoveryDisabled Определяем кастомную ошибку
var ErrDiscoveryDisabled = errors.New("service discovery is disabled in config")

type ServiceConfig struct {
	Host string
	Port int
}

type ServiceDiscovery interface {
	GetService(name string) (*ServiceConfig, error)
}

// ConfigResolver Stateless Resolver. Структура остается для реализации интерфейса, но без полей
type ConfigResolver struct{}

func NewConfigResolver() *ConfigResolver {
	return &ConfigResolver{}
}

func (r *ConfigResolver) GetService(name string) (*ServiceConfig, error) {
	// Если Discovery выключен — возвращаем специальную ошибку ErrDiscoveryDisabled
	if !viper.GetBool("discovery.enabled") {
		return nil, ErrDiscoveryDisabled
	}

	// Если включен — идем в Consul
	return r.queryConsul(name)
}

// queryConsul выполняет запрос к API Consul для поиска адреса сервиса
func (r *ConfigResolver) queryConsul(name string) (*ServiceConfig, error) {
	// Создаем стандартный конфиг клиента Consul
	config := api.DefaultConfig()

	// Берем адрес и токен из нашего hydro.yaml
	config.Address = viper.GetString("discovery.consul_addr")
	config.Token = viper.GetString("discovery.consul_token")

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	// Запрашиваем только здоровые (passing) экземпляры сервиса
	entries, _, err := client.Health().Service(name, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("consul query error for service %s: %w", name, err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("service %s not found or critical", name)
	}

	// Возвращаем реальный IP
	return &ServiceConfig{
		Host: entries[0].Service.Address, // Берем из первого элемента массива
		Port: entries[0].Service.Port,
	}, nil
}
