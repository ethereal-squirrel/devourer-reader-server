class RateLimiter {
  private queue: (() => Promise<void>)[] = [];
  private processing = false;
  private lastRequestTime = 0;
  private requestsThisPeriod = 0;
  private periodStart = Date.now();
  private requestsPerPeriod: number;
  private periodDuration: number;
  private minRequestInterval: number;

  constructor(
    requestsPerPeriod: number,
    periodDuration: number,
    minRequestInterval: number = 400
  ) {
    this.requestsPerPeriod = requestsPerPeriod;
    this.periodDuration = periodDuration;
    this.minRequestInterval = minRequestInterval;
  }

  private async processQueue() {
    if (this.processing) return;
    this.processing = true;

    while (this.queue.length > 0) {
      const now = Date.now();

      if (now - this.periodStart >= this.periodDuration) {
        this.requestsThisPeriod = 0;
        this.periodStart = now;
      }

      if (this.requestsThisPeriod >= this.requestsPerPeriod) {
        const waitTime = this.periodStart + this.periodDuration - now;
        await new Promise((resolve) => setTimeout(resolve, waitTime));
        continue;
      }

      if (now - this.lastRequestTime < this.minRequestInterval) {
        await new Promise((resolve) =>
          setTimeout(resolve, this.minRequestInterval)
        );
        continue;
      }

      const request = this.queue.shift();
      if (request) {
        this.lastRequestTime = Date.now();
        this.requestsThisPeriod++;
        await request();
      }
    }

    this.processing = false;
  }

  async schedule<T>(fn: () => Promise<T>): Promise<T> {
    return new Promise((resolve, reject) => {
      this.queue.push(async () => {
        try {
          const result = await fn();
          resolve(result);
        } catch (error) {
          reject(error);
        }
      });
      this.processQueue();
    });
  }
}

export const jikanLimiter = new RateLimiter(45, 60000);
export const metronLimiter = new RateLimiter(30, 60000);
export const googleBooksLimiter = new RateLimiter(30, 60000);
export const openLibraryLimiter = new RateLimiter(30, 60000);
export const comicVineLimiter = new RateLimiter(200, 3600000);
