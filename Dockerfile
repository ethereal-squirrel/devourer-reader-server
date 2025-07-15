FROM node:22-alpine

RUN apk add --no-cache \
    python3 \
    py3-setuptools \
    py3-pip \
    make \
    g++ \
    cairo-dev \
    jpeg-dev \
    pango-dev \
    musl-dev \
    giflib-dev \
    pixman-dev \
    pangomm-dev \
    libjpeg-turbo-dev \
    freetype-dev

WORKDIR /app

COPY package*.json ./

RUN npm install -g npm@latest && \
    npm i && \
    npm cache clean --force

COPY . .

RUN npx prisma generate && npx tsc

EXPOSE 9024

CMD ["node", "dist/index.js"]