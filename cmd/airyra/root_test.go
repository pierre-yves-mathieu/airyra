package main

import (
	"bytes"
	"testing"
)

func TestRootCmd_Exists(t *testing.T) {
	if rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}
}

func TestRootCmd_Use(t *testing.T) {
	if rootCmd.Use != "airyra" {
		t.Errorf("rootCmd.Use = %s, expected airyra", rootCmd.Use)
	}
}

func TestRootCmd_HasJSONFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("json")
	if flag == nil {
		t.Error("rootCmd should have --json flag")
	}
}

func TestRootCmd_JSONFlagDefault(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("json")
	if flag.DefValue != "false" {
		t.Errorf("--json flag default = %s, expected false", flag.DefValue)
	}
}

func TestRootCmd_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Errorf("rootCmd.Execute() returned error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Help output should not be empty")
	}
}
