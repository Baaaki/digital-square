# Development stage - no build, just run dev server
FROM oven/bun:1.3-alpine

WORKDIR /app

# Copy package files
COPY package.json bun.lock ./

# Install ALL dependencies (including devDependencies for TypeScript)
RUN bun install --frozen-lockfile

# Copy source code
COPY . .

# Expose port
EXPOSE 3000

# Set environment variables
# Note: NEXT_PUBLIC_* vars are injected at runtime from docker-compose.yml
ENV NODE_ENV=development

# Start development server
CMD ["bun", "run", "dev"]
