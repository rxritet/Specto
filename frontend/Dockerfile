# Этап сборки frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Этап раздачи статики через nginx
FROM nginx:alpine
COPY --from=frontend-builder /app/dist/frontend /usr/share/nginx/html
# Копируем конфиг nginx для SPA, если есть (или используем дефолтный)
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
