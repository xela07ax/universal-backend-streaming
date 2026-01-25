package discovery

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDiscoveryFallback(t *testing.T) {
	// Имитируем выключенный Discovery
	viper.Set("discovery.enabled", false)

	sd := NewConfigResolver()
	conf, err := sd.GetService("db-service")

	// Проверяем результат
	assert.Nil(t, conf)
	// В режиме false мы ожидаем ошибку ErrDiscoveryDisabled, которую BuildDSN обработает и возьмет статику
	assert.ErrorIs(t, err, ErrDiscoveryDisabled)
}
