package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	detach bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start KubeOrchestra services",
	Long: `Start KubeOrchestra services based on your initialization:
- If no repos cloned: runs from Docker images
- If UI cloned: runs UI locally with hot reload, Core from image
- If Core cloned: runs Core locally with hot reload, UI from image
- If both cloned: runs both locally with hot reload`,
	RunE: runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&detach, "detach", "d", false, "run services in background")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	if err := validateDockerCompose(); err != nil {
		return err
	}

	// detect what was initialized
	uiLocal := dirExists("./ui")
	coreLocal := dirExists("./core")

	fmt.Println("🚀 starting kubeorchestra services...")
	
	// determine which compose file to use
	var composeFile string
	
	if !uiLocal && !coreLocal {
		// production mode - everything from images
		fmt.Println("   mode: production (using docker images)")
		composeFile = "docker/docker-compose.prod.yml"
	} else if uiLocal && coreLocal {
		// full development mode
		fmt.Println("   mode: development (both local)")
		composeFile = "docker/docker-compose.dev.yml"
	} else if uiLocal {
		// ui development only
		fmt.Println("   mode: ui development (ui local, core from image)")
		composeFile = "docker/docker-compose.hybrid-ui.yml"
	} else {
		// core development only
		fmt.Println("   mode: core development (core local, ui from image)")
		composeFile = "docker/docker-compose.hybrid-core.yml"
	}
	
	// check if compose file exists
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		fmt.Printf("⚠️  compose file %s not found. creating default configuration...\n", composeFile)
		if err := createDefaultComposeFiles(); err != nil {
			return fmt.Errorf("failed to create compose files: %w", err)
		}
	}
	
	// build docker-compose command
	args = []string{"-f", composeFile, "up"}
	
	if detach {
		args = append(args, "-d")
	}
	
	// execute docker-compose
	composeCmd := exec.Command("docker-compose", args...)
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr
	composeCmd.Stdin = os.Stdin
	
	fmt.Println("   running: docker-compose", joinArgs(args))
	
	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}
	
	if detach {
		fmt.Println("✅ services started in background")
		fmt.Println("   view logs: orchcli logs")
		fmt.Println("   stop services: orchcli stop")
	}
	
	return nil
}


func createDefaultComposeFiles() error {
	// create docker directory if it doesn't exist
	if err := os.MkdirAll("docker", 0755); err != nil {
		return err
	}
	
	// create production compose file
	prodCompose := `version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: kubeorchestra
      POSTGRES_USER: kubeorchestra
      POSTGRES_PASSWORD: kubeorchestra
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  core:
    image: ghcr.io/kubeorchestra/core:latest
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_NAME: kubeorchestra
      DB_USER: kubeorchestra
      DB_PASSWORD: kubeorchestra
    depends_on:
      - postgres

  ui:
    image: ghcr.io/kubeorchestra/ui:latest
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
    depends_on:
      - core

volumes:
  postgres_data:
`
	
	if err := os.WriteFile("docker/docker-compose.prod.yml", []byte(prodCompose), 0644); err != nil {
		return err
	}
	
	// create development compose file
	devCompose := `version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: kubeorchestra
      POSTGRES_USER: kubeorchestra
      POSTGRES_PASSWORD: kubeorchestra
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  core:
    build:
      context: ../core
      dockerfile: Dockerfile.dev
    volumes:
      - ../core:/app
      - /app/vendor
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_NAME: kubeorchestra
      DB_USER: kubeorchestra
      DB_PASSWORD: kubeorchestra
    depends_on:
      - postgres
    command: air

  ui:
    build:
      context: ../ui
      dockerfile: Dockerfile.dev
    volumes:
      - ../ui:/app
      - /app/node_modules
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
    depends_on:
      - core
    command: npm run dev

volumes:
  postgres_data:
`
	
	if err := os.WriteFile("docker/docker-compose.dev.yml", []byte(devCompose), 0644); err != nil {
		return err
	}
	
	// create hybrid compose files
	hybridUICompose := `version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: kubeorchestra
      POSTGRES_USER: kubeorchestra
      POSTGRES_PASSWORD: kubeorchestra
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  core:
    image: ghcr.io/kubeorchestra/core:latest
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_NAME: kubeorchestra
      DB_USER: kubeorchestra
      DB_PASSWORD: kubeorchestra
    depends_on:
      - postgres

  ui:
    build:
      context: ../ui
      dockerfile: Dockerfile.dev
    volumes:
      - ../ui:/app
      - /app/node_modules
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
    depends_on:
      - core
    command: npm run dev

volumes:
  postgres_data:
`
	
	if err := os.WriteFile("docker/docker-compose.hybrid-ui.yml", []byte(hybridUICompose), 0644); err != nil {
		return err
	}
	
	hybridCoreCompose := `version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: kubeorchestra
      POSTGRES_USER: kubeorchestra
      POSTGRES_PASSWORD: kubeorchestra
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  core:
    build:
      context: ../core
      dockerfile: Dockerfile.dev
    volumes:
      - ../core:/app
      - /app/vendor
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_NAME: kubeorchestra
      DB_USER: kubeorchestra
      DB_PASSWORD: kubeorchestra
    depends_on:
      - postgres
    command: air

  ui:
    image: ghcr.io/kubeorchestra/ui:latest
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
    depends_on:
      - core

volumes:
  postgres_data:
`
	
	if err := os.WriteFile("docker/docker-compose.hybrid-core.yml", []byte(hybridCoreCompose), 0644); err != nil {
		return err
	}
	
	fmt.Println("✅ created default docker-compose files")
	return nil
}