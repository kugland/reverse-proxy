package main

import "testing"

func TestConfigurePaths(t *testing.T) {
	got, err := ConfigurePaths()
	if err != nil {
		t.Error("Falhou err", got, err)
	}
}
