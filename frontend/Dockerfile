# --- Build ---
FROM node:24-alpine AS build

WORKDIR /app

# install dependencies first for better layer caching
COPY package.json package-lock.json ./
RUN npm ci

# build the site
COPY . .
RUN npm run build


# --- Serve ---
# nginx serves the built site and passes /api on to the backend, so the site
# and the API share one origin and the session cookies need no cross-origin
# setup, as Vite's proxy does in development.
FROM nginx:1.27-alpine

# the gateway this site talks to; another one needs no rebuild
ENV BACKEND_ORIGIN=http://backend:8000
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
