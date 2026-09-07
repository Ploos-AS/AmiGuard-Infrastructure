package main

import "testing"

func TestEnvFallback(t *testing.T) {
    t.Setenv("AMIGUARD_TEST_VALUE", "")
    if got := env("AMIGUARD_TEST_VALUE", "fallback"); got != "fallback" {
        t.Fatalf("got %q", got)
    }
    t.Setenv("AMIGUARD_TEST_VALUE", "set")
    if got := env("AMIGUARD_TEST_VALUE", "fallback"); got != "set" {
        t.Fatalf("got %q", got)
    }
}
