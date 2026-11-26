FROM node:24-alpine
WORKDIR /app
COPY package.json .
RUN npm install --omit=dev
COPY . .
ENV DB_PATH=/data/products.db
RUN mkdir -p /data
VOLUME /data
EXPOSE 8003
CMD ["node", "index.js"]
