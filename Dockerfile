FROM node:22-alpine

RUN apk add --no-cache \
    python3 \
    py3-setuptools \
    py3-pip \
    make \
    g++ \
    cairo-dev \
    cairo \
    jpeg-dev \
    pango-dev \
    pango \
    musl-dev \
    giflib-dev \
    giflib \
    pixman-dev \
    pixman \
    pangomm-dev \
    libjpeg-turbo-dev \
    libjpeg-turbo \
    freetype-dev \
    freetype

WORKDIR /app

COPY package*.json ./

RUN npm install -g npm@latest rimraf node-pre-gyp node-gyp && \
    npm i && \
    npm cache clean --force

COPY . .

RUN npx prisma generate && npx tsc

EXPOSE 9024

CMD ["node", "dist/index.js"]