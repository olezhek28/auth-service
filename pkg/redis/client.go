package redis

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

// Config содержит настройки подключения к Redis
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// NewRedisPool создает пул соединений с Redis
func NewRedisPool(cfg Config) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}

			if cfg.Password != "" {
				if _, err := c.Do("AUTH", cfg.Password); err != nil {
					c.Close()
					return nil, err
				}
			}

			if _, err := c.Do("SELECT", cfg.DB); err != nil {
				c.Close()
				return nil, err
			}

			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
