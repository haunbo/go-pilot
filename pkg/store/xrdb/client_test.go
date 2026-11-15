package xrdb

import "testing"

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := cfg.Build(); err == nil {
		t.Fatal("empty dsn should fail")
	}
	cfg.DSN = "user:pass@tcp(127.0.0.1:3306)/demo"
	cfg.Type = "oracle"
	if _, err := cfg.Build(); err == nil {
		t.Fatal("unsupported type should fail")
	}
}

func TestSQLiteBuild(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Type = SQLite
	cfg.DSN = "file:go_pilot_test?mode=memory&cache=shared"
	cfg.MaxOpenConns = 1
	cfg.MaxIdleConns = 1
	client, err := cfg.Build()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	type record struct {
		ID   uint
		Name string
	}
	if err := client.AutoMigrate(&record{}); err != nil {
		t.Fatal(err)
	}
	if err := client.Gorm().Create(&record{Name: "demo"}).Error; err != nil {
		t.Fatal(err)
	}
	var got record
	if err := client.Gorm().First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "demo" {
		t.Fatalf("name = %q", got.Name)
	}
}
