# Start infrastructure services
up-services:
	docker-compose -f docker-compose.services.yml up -d

# Build and run the app only
up-app:
	docker-compose -f docker-compose.app.yml up --build

# Stop everything
down:
	docker-compose -f docker-compose.services.yml down
	docker-compose -f docker-compose.app.yml down

# Rebuild app only (without starting)
build-app:
	docker-compose -f docker-compose.app.yml build

# Restart app only (no rebuild)
restart-app:
	docker-compose -f docker-compose.app.yml restart
