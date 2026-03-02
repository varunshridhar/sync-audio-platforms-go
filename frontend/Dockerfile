# Build stage installs dependencies and produces optimized Next.js output.
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

# Runtime stage serves compiled app in production mode.
FROM node:20-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
STOPSIGNAL SIGTERM
CMD ["node", "node_modules/next/dist/bin/next", "start", "-p", "3000"]

