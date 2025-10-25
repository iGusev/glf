package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/igusev/glf/internal/cache"
	"github.com/igusev/glf/internal/history"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/model"
)

func main() {
	// Create demo cache directory in demo/data/glf
	demoDir := "demo/data/glf"
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create demo dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generating fake data in: %s\n", demoDir)

	// Generate fake projects
	projects := []model.Project{
		// Backend services
		{
			Path:        "backend/api/user-service",
			Name:        "user-service",
			Description: "User authentication and profile management service with JWT tokens",
			Starred:     true,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/api/payment-gateway",
			Name:        "payment-gateway",
			Description: "Payment processing and transaction management with Stripe integration",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/api/notification-service",
			Name:        "notification-service",
			Description: "Multi-channel notification delivery system (email, SMS, push)",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/api/search-engine",
			Name:        "search-engine",
			Description: "Elasticsearch-based full-text search service with fuzzy matching",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/api/order-management",
			Name:        "order-management",
			Description: "Order processing and inventory management system",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/api/recommendation-engine",
			Name:        "recommendation-engine",
			Description: "ML-powered product recommendation service",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/workers/email-queue",
			Name:        "email-queue",
			Description: "Background job processor for email delivery with Redis queue",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "backend/workers/image-processor",
			Name:        "image-processor",
			Description: "Async image optimization and thumbnail generation worker",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},

		// Frontend applications
		{
			Path:        "frontend/web/dashboard",
			Name:        "dashboard",
			Description: "Admin dashboard with real-time analytics and monitoring",
			Starred:     true,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "frontend/web/customer-portal",
			Name:        "customer-portal",
			Description: "Customer-facing web application built with React and TypeScript",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "frontend/web/marketing-site",
			Name:        "marketing-site",
			Description: "Public marketing website with Next.js and Tailwind CSS",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "frontend/mobile/ios-app",
			Name:        "ios-app",
			Description: "Native iOS application written in Swift with SwiftUI",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "frontend/mobile/android-app",
			Name:        "android-app",
			Description: "Native Android application with Kotlin and Jetpack Compose",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "frontend/mobile/react-native-app",
			Name:        "react-native-app",
			Description: "Cross-platform mobile app with React Native",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},

		// Infrastructure
		{
			Path:        "infrastructure/terraform/aws",
			Name:        "aws",
			Description: "AWS infrastructure as code with Terraform modules",
			Starred:     true,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "infrastructure/terraform/gcp",
			Name:        "gcp",
			Description: "Google Cloud Platform infrastructure with Terraform",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "infrastructure/kubernetes/manifests",
			Name:        "manifests",
			Description: "Kubernetes deployment manifests and Helm charts",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "infrastructure/kubernetes/operators",
			Name:        "operators",
			Description: "Custom Kubernetes operators for application management",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "infrastructure/docker/base-images",
			Name:        "base-images",
			Description: "Base Docker images for all services with security hardening",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},

		// DevOps
		{
			Path:        "devops/ci-cd/jenkins-pipelines",
			Name:        "jenkins-pipelines",
			Description: "CI/CD pipeline configurations and shared libraries",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "devops/ci-cd/github-actions",
			Name:        "github-actions",
			Description: "Reusable GitHub Actions workflows",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "devops/monitoring/prometheus",
			Name:        "prometheus",
			Description: "Prometheus monitoring stack with Grafana dashboards",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "devops/monitoring/elk-stack",
			Name:        "elk-stack",
			Description: "Elasticsearch, Logstash, Kibana logging infrastructure",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "devops/security/vault-config",
			Name:        "vault-config",
			Description: "HashiCorp Vault configuration for secrets management",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},

		// Data & Analytics
		{
			Path:        "data/analytics/warehouse",
			Name:        "warehouse",
			Description: "Data warehouse and ETL processes with Airflow",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "data/analytics/reports",
			Name:        "reports",
			Description: "Business intelligence reports and dashboards",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "data/ml/training-pipelines",
			Name:        "training-pipelines",
			Description: "Machine learning model training pipelines with MLflow",
			Starred:     false,
			Archived:    false,
			Member:      false,
		},
		{
			Path:        "data/ml/model-serving",
			Name:        "model-serving",
			Description: "ML model serving infrastructure with TensorFlow Serving",
			Starred:     false,
			Archived:    false,
			Member:      false,
		},

		// Libraries & Tools
		{
			Path:        "libraries/common/utils",
			Name:        "utils",
			Description: "Shared utility functions and helpers",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "libraries/ui-components",
			Name:        "ui-components",
			Description: "Reusable React component library with Storybook",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "tools/cli/developer-tools",
			Name:        "developer-tools",
			Description: "Internal CLI tools for developers",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "tools/scripts/automation",
			Name:        "automation",
			Description: "Automation scripts for common tasks",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "tools/generators/project-scaffolding",
			Name:        "project-scaffolding",
			Description: "Project template generator with best practices",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},

		// Documentation & Archived
		{
			Path:        "docs/architecture/decisions",
			Name:        "decisions",
			Description: "Architecture Decision Records (ADRs)",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "docs/api/specifications",
			Name:        "specifications",
			Description: "OpenAPI specifications for all services",
			Starred:     false,
			Archived:    false,
			Member:      true,
		},
		{
			Path:        "archived/legacy/old-monolith",
			Name:        "old-monolith",
			Description: "Legacy monolithic application (archived)",
			Starred:     false,
			Archived:    true,
			Member:      false,
		},
		{
			Path:        "archived/prototypes/blockchain-poc",
			Name:        "blockchain-poc",
			Description: "Blockchain proof of concept (archived)",
			Starred:     false,
			Archived:    true,
			Member:      false,
		},
	}

	// Save projects cache
	cacheFile := filepath.Join(demoDir, "projects.json")
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal projects: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write cache: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Created projects cache (%d projects)\n", len(projects))

	// Create description index
	indexPath := filepath.Join(demoDir, "description.bleve")
	descIndex, _, err := index.NewDescriptionIndexWithAutoRecreate(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create index: %v\n", err)
		os.Exit(1)
	}
	defer descIndex.Close()

	// Index all projects
	var docs []index.DescriptionDocument
	for _, proj := range projects {
		docs = append(docs, index.DescriptionDocument{
			ProjectPath: proj.Path,
			ProjectName: proj.Name,
			Description: proj.Description,
			Starred:     proj.Starred,
			Archived:    proj.Archived,
			Member:      proj.Member,
		})
	}

	if err := descIndex.AddBatch(docs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to index projects: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Created search index\n")

	// Create fake history
	historyPath := filepath.Join(demoDir, "history.gob")
	hist := history.New(historyPath)

	// Simulate usage history
	now := time.Now()
	selections := []struct {
		path  string
		query string
		count int
		days  int
	}{
		{"backend/api/user-service", "user api", 25, 0},
		{"frontend/web/dashboard", "dashboard", 18, 0},
		{"infrastructure/terraform/aws", "terraform aws", 12, 0},
		{"backend/api/payment-gateway", "payment", 10, 0},
		{"frontend/mobile/ios-app", "ios mobile", 8, 1},
		{"devops/monitoring/prometheus", "monitoring", 7, 1},
		{"backend/api/notification-service", "notification", 5, 1},
		{"data/analytics/warehouse", "warehouse", 4, 2},
		{"libraries/ui-components", "ui components", 3, 2},
		{"tools/cli/developer-tools", "dev tools", 2, 3},
	}

	for _, sel := range selections {
		for i := 0; i < sel.count; i++ {
			hist.RecordSelectionWithQuery(sel.query, sel.path)
		}
	}

	if err := hist.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save history: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Created search history\n")

	// Save cache metadata
	cacheManager := cache.New(demoDir)
	if err := cacheManager.SaveLastSyncTime(now); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save sync time: %v\n", err)
		os.Exit(1)
	}
	if err := cacheManager.SaveLastFullSyncTime(now); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save full sync time: %v\n", err)
		os.Exit(1)
	}
	if err := cacheManager.SaveUsername("demo-user"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save username: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Created cache metadata\n")

	fmt.Printf("\n✅ Demo data generated successfully!\n\n")
	fmt.Printf("To use with GLF, set XDG_CACHE_HOME:\n")
	fmt.Printf("  export XDG_CACHE_HOME=$(pwd)/scripts/demo-data\n")
	fmt.Printf("  glf\n\n")
	fmt.Printf("Demo directory: %s\n", demoDir)
}
