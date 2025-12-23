.PHONY: help up down logs build rebuild clean seed restart ps

# Default target
help:
	@echo "Digital Square - Docker Commands"
	@echo ""
	@echo "Available commands:"
	@echo "  make up          - Start all services (postgres, redis, backend, frontend)"
	@echo "  make down        - Stop all services"
	@echo "  make logs        - Show logs from all services"
	@echo "  make build       - Build all Docker images"
	@echo "  make rebuild     - Rebuild and restart all services"
	@echo "  make clean       - Stop services and remove volumes (WARNING: deletes data)"
	@echo "  make seed        - Create admin user in database"
	@echo "  make restart     - Restart all services"
	@echo "  make ps          - Show running containers"
	@echo ""

# Start all services
up:
	@echo "Starting Digital Square services..."
	sudo docker compose up -d
	@echo ""
	@echo "Services started successfully!"
	@echo "Frontend: http://localhost:3001"
	@echo "Backend:  http://localhost:9090"
	@echo ""
	@echo "Run 'make logs' to view logs"
	@echo "Run 'make seed' to create admin user"

# Stop all services
down:
	@echo "Stopping Digital Square services..."
	sudo docker compose down
	@echo "Services stopped."

# Show logs from all services
logs:
	sudo docker compose logs -f

# Show logs from specific service
logs-backend:
	sudo docker compose logs -f backend

logs-frontend:
	sudo docker compose logs -f frontend

logs-postgres:
	sudo docker compose logs -f postgres

logs-redis:
	sudo docker compose logs -f redis

# Build all images
build:
	@echo "Building Docker images..."
	sudo docker compose build
	@echo "Build completed."

# Rebuild and restart all services
rebuild:
	@echo "Rebuilding and restarting services..."
	sudo docker compose down
	sudo docker compose build
	sudo docker compose up -d
	@echo "Services rebuilt and restarted."

# Clean everything (removes volumes)
clean:
	@echo "WARNING: This will delete all data (database, redis cache)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		sudo docker compose down -v; \
		echo "All services stopped and data removed."; \
	else \
		echo "Cancelled."; \
	fi

# Create admin user
seed:
	@echo "Creating admin user..."
	sudo docker compose exec backend ./seed
	@echo ""
	@echo "Admin user created!"
	@echo "Email: admin@digitalsquare.com"
	@echo "Password: Admin123SecurePassword"

# Restart all services
restart:
	@echo "Restarting services..."
	sudo docker compose restart
	@echo "Services restarted."

# Restart specific service
restart-backend:
	sudo docker compose restart backend

restart-frontend:
	sudo docker compose restart frontend

# Show running containers
ps:
	sudo docker compose ps

# Enter backend container shell
shell-backend:
	sudo docker compose exec backend sh

# Enter frontend container shell
shell-frontend:
	sudo docker compose exec frontend sh

# Enter postgres container
shell-postgres:
	sudo docker compose exec postgres psql -U postgres -d digitalsquare

# Enter redis container
shell-redis:
	sudo docker compose exec redis redis-cli
