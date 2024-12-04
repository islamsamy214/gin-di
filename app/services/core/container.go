package core

import (
	"log"
)

type Container struct {
	services map[string]interface{}
}

func NewContainer() *Container {
	return &Container{services: make(map[string]interface{})}
}

func (c *Container) Bind(key string, value interface{}) {
	c.services[key] = value
}

func (c *Container) Resolve(key string) interface{} {
	service, exists := c.services[key]
	if !exists {
		log.Printf("Service %s not found", key)
	}
	return service
}
