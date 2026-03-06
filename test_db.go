package main

import (
	"log"

	"github.com/Lovealone1/nex21-api/internal/platform/config"
	"github.com/Lovealone1/nex21-api/internal/platform/db"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS public.products (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE, name TEXT NOT NULL, description TEXT NULL, sku TEXT NULL, price NUMERIC(12,2) NOT NULL DEFAULT 0, currency TEXT NOT NULL DEFAULT 'COP', quantity INT NOT NULL DEFAULT 0, is_active BOOLEAN NOT NULL DEFAULT true, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());`,
		`CREATE INDEX IF NOT EXISTS products_tenant_id_idx ON public.products (tenant_id);`,
		`CREATE INDEX IF NOT EXISTS products_tenant_name_idx ON public.products (tenant_id, lower(name));`,
		`CREATE UNIQUE INDEX IF NOT EXISTS products_tenant_sku_idx ON public.products (tenant_id, sku);`,
		`CREATE TABLE IF NOT EXISTS public.services (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE, name TEXT NOT NULL, description TEXT NULL, sku TEXT NULL, price NUMERIC(12,2) NOT NULL DEFAULT 0, currency TEXT NOT NULL DEFAULT 'COP', is_active BOOLEAN NOT NULL DEFAULT true, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());`,
		`CREATE INDEX IF NOT EXISTS services_tenant_id_idx ON public.services (tenant_id);`,
		`CREATE INDEX IF NOT EXISTS services_tenant_name_idx ON public.services (tenant_id, lower(name));`,
		`CREATE UNIQUE INDEX IF NOT EXISTS services_tenant_sku_idx ON public.services (tenant_id, sku);`,
	}

	for i, q := range queries {
		err := database.Exec(q).Error
		if err != nil {
			log.Fatalf("Error on query %d: %v", i, err)
		}
		log.Printf("Query %d success", i)
	}
}
