package main

import (
	"go-rabbitmq-consumers/utils"
	"testing"
)

func TestMain(t *testing.T) {
	t.Log(utils.GetUUID())
}
