package main

import (
	"testing"
)

func setup() {
	connect("127.0.0.1:13203")
	setNames("oss", []string{"1", "2", "3"})
	setNames("bss", []string{"a", "b", "c"})
}
func teardown() {
	disconnect()
}
func TestSetNames(t *testing.T) {
	setup()
	defer teardown()
	setNames("oss", []string{"x", "y", "z"})
	m := toCollection(getConfig())
	if m["x"] != "oss" {
		t.Error("setNames")
	}
}
func TestAddNames(t *testing.T) {
	setup()
	defer teardown()
	addNames("oss", []string{"x", "y"})
	m := toCollection(getConfig())
	if m["y"] != "oss" {
		t.Error("addNames")
	}
}
func TestDelNames(t *testing.T) {
	setup()
	defer teardown()
	addNames("bss", []string{"vvv", "zzz"})

	delNames([]string{"1", "vvv"})
	m := toCollection(getConfig())
	if _, ok := m["123"]; ok {
		t.Error("delNames")
	}
}
